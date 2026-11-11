#!/usr/bin/env python3
"""Convert MySQL/MariaDB SQL dumps or SQLite files into PostgreSQL SQL.

Uses the Python standard library. The result is intended for an empty database.
Unsupported statements and schema features cause explicit errors.
"""

from __future__ import annotations

import argparse
import hashlib
import os
import re
import sqlite3
import sys
import tempfile
from collections import defaultdict
from pathlib import Path
from typing import TextIO


class ConversionError(ValueError):
    pass


def ident(value: str) -> str:
    if not value or '\x00' in value or len(value.encode('utf-8')) > 63:
        raise ConversionError(f'Invalid PostgreSQL identifier: {value!r}')
    return '"' + value.replace('"', '""') + '"'


def literal(value: str) -> str:
    if '\x00' in value:
        raise ConversionError('NUL byte in a text value; use a binary column')
    return "'" + value.replace("'", "''") + "'"


def unquote_mysql(value: str) -> str:
    if len(value) < 2 or value[0] not in ("'", '"') or value[-1] != value[0]:
        raise ConversionError(f'Expected MySQL quoted string: {value[:80]!r}')
    out = []
    i = 1
    mapping = {'0': '\x00', 'b': '\b', 'n': '\n', 'r': '\r', 't': '\t', 'Z': '\x1a'}
    while i < len(value) - 1:
        c = value[i]
        if c == '\\' and i + 1 < len(value) - 1:
            i += 1
            out.append(mapping.get(value[i], value[i]))
        elif c == value[0] and i + 1 < len(value) - 1 and value[i + 1] == c:
            out.append(c)
            i += 1
        else:
            out.append(c)
        i += 1
    return ''.join(out)


def split_top(text: str, delimiter: str = ',') -> list[str]:
    """Split outside SQL strings, identifier quotes and parentheses."""
    parts = []
    state = ''
    depth = 0
    start = 0
    i = 0
    while i < len(text):
        c = text[i]
        if state:
            if c == '\\' and state in ("'", '"') and i + 1 < len(text):
                i += 2
                continue
            if c == state:
                if i + 1 < len(text) and text[i + 1] == state:
                    i += 2
                    continue
                state = ''
        elif c in ("'", '"', '`'):
            state = c
        elif c == '(':
            depth += 1
        elif c == ')':
            depth -= 1
            if depth < 0:
                raise ConversionError('Unexpected closing parenthesis')
        elif c == delimiter and depth == 0:
            parts.append(text[start:i].strip())
            start = i + 1
        i += 1
    if state or depth != 0:
        raise ConversionError('Unterminated SQL string or parentheses')
    parts.append(text[start:].strip())
    return parts


def sql_statements(stream: TextIO):
    """Yield statements, ignoring comments only where they are outside strings."""
    buffer = []
    state = ''
    in_block = False
    i = 0
    while True:
        line = stream.readline()
        if not line:
            break
        if re.match(r'^\s*DELIMITER\b', line, re.I):
            raise ConversionError('Stored routines with DELIMITER are unsupported')
        i = 0
        while i < len(line):
            c = line[i]
            nxt = line[i + 1] if i + 1 < len(line) else ''
            if in_block:
                if c == '*' and nxt == '/':
                    in_block = False
                    i += 2
                else:
                    i += 1
                continue
            if not state and c == '/' and nxt == '*':
                in_block = True
                i += 2
                continue
            if not state and (c == '#' or (c == '-' and nxt == '-' and (i + 2 >= len(line) or line[i + 2].isspace()))):
                break
            if state:
                buffer.append(c)
                if c == '\\' and state in ("'", '"') and nxt:
                    buffer.append(nxt)
                    i += 2
                    continue
                if c == state:
                    if nxt == state:
                        buffer.append(nxt)
                        i += 2
                        continue
                    state = ''
            elif c in ("'", '"', '`'):
                state = c
                buffer.append(c)
            elif c == ';':
                stmt = ''.join(buffer).strip()
                if stmt:
                    yield stmt
                buffer = []
            else:
                buffer.append(c)
            i += 1
        buffer.append('\n')
    if in_block or state:
        raise ConversionError('Unterminated SQL quote or comment')
    if ''.join(buffer).strip():
        raise ConversionError('Unterminated SQL statement; expected semicolon')


MYSQL_NAME = r'(?:`((?:``|[^`])+)`|([A-Za-z_][A-Za-z0-9_]*))'


def mysql_name(text: str) -> str:
    match = re.fullmatch(MYSQL_NAME, text.strip(), re.S)
    if not match:
        raise ConversionError(f'Unsupported table or column name: {text[:100]!r}')
    return (match.group(1) or match.group(2)).replace('``', '`')


def mysql_cols(text: str) -> list[str]:
    result = []
    for item in split_top(text):
        match = re.fullmatch(r'\s*(' + MYSQL_NAME + r')\s*(?:\(\d+\))?\s*(?:ASC|DESC)?\s*', item, re.I | re.S)
        if not match:
            raise ConversionError(f'Unsupported index expression: {item[:90]!r}')
        col = mysql_name(match.group(1))
        prefix = re.search(r'\((\d+)\)\s*(?:ASC|DESC)?\s*$', item, re.I)
        expr = f'LEFT({ident(col)}, {int(prefix.group(1))})' if prefix else ident(col)
        direction = re.search(r'\s+(ASC|DESC)\s*$', item, re.I)
        if direction:
            expr += ' ' + direction.group(1).upper()
        result.append(expr)
    return result


def take_parenthesized(text: str) -> tuple[str, str]:
    text = text.lstrip()
    if not text.startswith('('):
        raise ConversionError('Expected opening parenthesis')
    # Reuse the tokenizer to find the first top-level closing parenthesis.
    depth = 0
    state = ''
    i = 0
    while i < len(text):
        c = text[i]
        if state:
            if c == '\\' and state in ("'", '"') and i + 1 < len(text):
                i += 2
                continue
            if c == state:
                if i + 1 < len(text) and text[i+1] == state:
                    i += 2
                    continue
                state = ''
        elif c in ("'", '"', '`'):
            state = c
        elif c == '(':
            depth += 1
        elif c == ')':
            depth -= 1
            if depth == 0:
                return text[1:i], text[i+1:].strip()
        i += 1
    raise ConversionError('Unclosed parenthesis')


def postgres_type(original: str) -> tuple[str, bool]:
    """Return (PG type, is_boolean); preserve unsigned integer ranges."""
    t = re.sub(r'\s+', ' ', original.strip().lower())
    unsigned = 'unsigned' in t
    t = re.sub(r'\s+unsigned\b', '', t).strip()
    m = re.fullmatch(r'([a-z]+)(?:\(([^)]*)\))?', t)
    if not m:
        raise ConversionError(f'Unsupported data type: {original!r}')
    base, args = m.groups()
    arg = f'({args})' if args else ''
    if base in ('bool', 'boolean') or (base == 'tinyint' and args == '1'):
        return 'BOOLEAN', True
    ints = {'tinyint': ('SMALLINT', 'SMALLINT'), 'smallint': ('SMALLINT', 'INTEGER'),
            'mediumint': ('INTEGER', 'INTEGER'), 'int': ('INTEGER', 'BIGINT'),
            'integer': ('INTEGER', 'BIGINT'), 'bigint': ('BIGINT', 'NUMERIC(20,0)')}
    if base in ints:
        return ints[base][1 if unsigned else 0], False
    if base in ('decimal', 'numeric', 'dec'):
        return 'NUMERIC' + arg, False
    if base == 'float':
        return 'REAL', False
    if base in ('double', 'real'):
        return 'DOUBLE PRECISION', False
    if base in ('varchar', 'char'):
        return ('VARCHAR' if base == 'varchar' else 'CHAR') + arg, False
    if base in ('tinytext', 'text', 'mediumtext', 'longtext', 'enum', 'set'):
        return 'TEXT', False
    if base in ('tinyblob', 'blob', 'mediumblob', 'longblob', 'binary', 'varbinary'):
        return 'BYTEA', False
    if base in ('datetime', 'timestamp'):
        return 'TIMESTAMP' + arg, False
    if base in ('date', 'time'):
        return base.upper() + arg, False
    if base == 'year':
        return 'SMALLINT', False
    if base == 'json':
        return 'JSONB', False
    if base == 'bit':
        return 'BIT' + arg, False
    raise ConversionError(f'Unsupported data type: {original!r}')


def pg_value(token: str, pgtype: str, boolean: bool = False) -> str:
    val = token.strip()
    if val.upper() == 'NULL':
        return 'NULL'
    if re.fullmatch(r"(?i)b'[01]+'", val):
        bits = val[2:-1]
        if boolean and len(bits) == 1:
            return 'TRUE' if bits == '1' else 'FALSE'
        return "B'" + bits + "'"
    if re.fullmatch(r'(?i)0x[0-9a-f]+', val):
        if pgtype != 'BYTEA':
            raise ConversionError(f'Hex value requires BYTEA: {val[:40]}')
        return f"decode('{val[2:]}', 'hex')"
    if val.startswith(("'", '"')):
        value = unquote_mysql(val)
        if pgtype == 'BYTEA':
            return f"decode('{value.encode('utf-8').hex()}', 'hex')"
        if value.startswith('0000-00-00') and pgtype.startswith(('DATE', 'TIMESTAMP')):
            raise ConversionError('Zero date cannot be represented in PostgreSQL')
        if boolean:
            if value not in ('0', '1'):
                raise ConversionError(f'Unexpected boolean value: {value!r}')
            return 'TRUE' if value == '1' else 'FALSE'
        return literal(value)
    if re.fullmatch(r'[+-]?(?:\d+(?:\.\d*)?|\.\d+)(?:[eE][+-]?\d+)?', val):
        if boolean:
            if val not in ('0', '1'):
                raise ConversionError(f'Unexpected boolean value: {val!r}')
            return 'TRUE' if val == '1' else 'FALSE'
        return val
    if val.upper() in ('TRUE', 'FALSE'):
        return val.upper()
    if re.fullmatch(r'(?i)(?:CURRENT_TIMESTAMP(?:\(\d+\))?|CURRENT_DATE|CURRENT_TIME|NOW\(\))', val):
        return 'CURRENT_TIMESTAMP' if val.upper() == 'NOW()' else val.upper()
    raise ConversionError(f'Unsupported SQL value: {val[:120]!r}')


def index_name(table: str, name: str, used: set[str]) -> str:
    candidate = name
    if len(candidate.encode('utf-8')) > 63 or candidate in used:
        candidate = f'{table}_{name}'
    if len(candidate.encode('utf-8')) > 63 or candidate in used:
        candidate = f'{name[:42]}_{hashlib.sha1((table + name).encode()).hexdigest()[:12]}'
    if candidate in used:
        raise ConversionError(f'Duplicate index name: {candidate}')
    used.add(candidate)
    return ident(candidate)


def convert_mysql(stream: TextIO, out: TextIO) -> tuple[int, int]:
    tables: dict[str, list[tuple[str, str, bool]]] = {}
    indexes: list[str] = []
    fks: list[str] = []
    sequences: list[tuple[str, str]] = []
    used_index_names: set[str] = set()
    rows_count = 0
    out.write('BEGIN;\nSET client_encoding = \'UTF8\';\nSET standard_conforming_strings = on;\n\n')
    for stmt in sql_statements(stream):
        stmt = stmt.strip()
        if re.match(r'(?is)^SET\s+', stmt) and 'NO_BACKSLASH_ESCAPES' in stmt.upper():
            raise ConversionError('NO_BACKSLASH_ESCAPES mode requires a separate conversion')
        if re.match(r'(?is)^(?:SET\s+|DROP\s+TABLE\b|LOCK\s+TABLES\b|UNLOCK\s+TABLES\b|START\s+TRANSACTION\b|BEGIN\b|COMMIT\b|USE\s+)', stmt):
            continue
        create = re.match(r'(?is)^CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(' + MYSQL_NAME + r')\s*(.*)$', stmt)
        if create:
            name = mysql_name(create.group(1))
            if name in tables:
                raise ConversionError(f'Duplicate table declaration: {name}')
            content, options = take_parenthesized(create.group(4))
            if re.search(r'(?is)\bPARTITION\s+BY\b', options):
                raise ConversionError(f'Partitioned tables require manual migration: {name}')
            if options and not re.match(r'(?is)^(?:ENGINE|AUTO_INCREMENT|DEFAULT\s+CHARSET|CHARSET|COLLATE|ROW_FORMAT|COMMENT|KEY_BLOCK_SIZE)\b', options):
                raise ConversionError(f'Unsupported CREATE TABLE options for {name}: {options[:100]}')
            cols: list[tuple[str, str, bool]] = []
            lines: list[str] = []
            primary: list[str] = []
            pending: list[tuple[bool, str, list[str]]] = []
            table_sequences: list[tuple[str, str]] = []
            for item in split_top(content):
                item = item.strip()
                if not item:
                    continue
                pk = re.match(r'(?is)^PRIMARY\s+KEY\s*', item)
                if pk:
                    content_pk, suffix = take_parenthesized(item[pk.end():])
                    if suffix and not re.fullmatch(r'(?is)USING\s+BTREE', suffix):
                        raise ConversionError(f'Unsupported PRIMARY KEY modifier: {suffix}')
                    primary = mysql_cols(content_pk)
                    continue
                key = re.match(r'(?is)^(UNIQUE\s+KEY|UNIQUE\s+INDEX|KEY|INDEX)\s+(' + MYSQL_NAME + r')\s*(.*)$', item)
                if key:
                    unique = key.group(1).upper().startswith('UNIQUE')
                    content_idx, suffix = take_parenthesized(key.group(5))
                    if suffix and not re.fullmatch(r'(?is)USING\s+BTREE', suffix):
                        raise ConversionError(f'Unsupported index modifier: {suffix}')
                    pending.append((unique, mysql_name(key.group(2)), mysql_cols(content_idx)))
                    continue
                fk = re.match(r'(?is)^(?:CONSTRAINT\s+(' + MYSQL_NAME + r')\s+)?FOREIGN\s+KEY\s*(.*)$', item)
                if fk:
                    fk_cols, remainder = take_parenthesized(fk.group(4))
                    ref = re.match(r'(?is)^REFERENCES\s+(' + MYSQL_NAME + r')\s*(.*)$', remainder)
                    if not ref:
                        raise ConversionError(f'Unsupported foreign key declaration: {item[:120]}')
                    ref_cols, actions = take_parenthesized(ref.group(4))
                    if actions and not re.fullmatch(r'(?is)(?:\s*ON\s+(?:DELETE|UPDATE)\s+(?:RESTRICT|CASCADE|SET\s+NULL|NO\s+ACTION|SET\s+DEFAULT))*', actions):
                        raise ConversionError(f'Unsupported foreign key actions: {actions}')
                    fks.append(f'ALTER TABLE {ident(name)} ADD FOREIGN KEY ({", ".join(mysql_cols(fk_cols))}) REFERENCES {ident(mysql_name(ref.group(1)))} ({", ".join(mysql_cols(ref_cols))}) {actions};')
                    continue
                if re.match(r'(?is)^(?:CHECK|CONSTRAINT|FULLTEXT|SPATIAL)\b', item):
                    raise ConversionError(f'Unsupported table constraint in {name}: {item[:100]}')
                col = re.match(r'(?is)^(' + MYSQL_NAME + r')\s+([a-z]+(?:\([^)]*\))?)(.*)$', item)
                if not col:
                    raise ConversionError(f'Unsupported column declaration in {name}: {item[:120]}')
                col_name = mysql_name(col.group(1))
                decl = col.group(5).strip()
                unsigned = re.match(r'(?is)^UNSIGNED\b', decl)
                coltype, boolean = postgres_type(col.group(4) + (' unsigned' if unsigned else ''))
                if unsigned:
                    decl = decl[unsigned.end():].strip()
                if re.search(r'(?i)\b(GENERATED|VIRTUAL|STORED|ON\s+UPDATE|CHECK|REFERENCES)\b', decl):
                    raise ConversionError(f'Unsupported column constraint in {name}.{col_name}')
                if re.search(r'(?i)\bUNIQUE\b', decl):
                    raise ConversionError(f'Inline UNIQUE requires a table-level index in {name}.{col_name}')
                auto = bool(re.search(r'(?i)\bAUTO_INCREMENT\b', decl))
                null = bool(re.search(r'(?i)\bNOT\s+NULL\b', decl))
                inline_pk = bool(re.search(r'(?i)\bPRIMARY\s+KEY\b', decl))
                default = re.search(r'(?is)\bDEFAULT\s+(NULL|(?:\'(?:\\.|\'\'|[^\'])*\')|(?:"(?:\\.|""|[^"])*")|[+-]?(?:\d+(?:\.\d*)?|\.\d+)|CURRENT_TIMESTAMP(?:\(\d+\))?|CURRENT_DATE|CURRENT_TIME|b\'[01]+\')', decl)
                if auto and coltype not in ('SMALLINT', 'INTEGER', 'BIGINT'):
                    seq = index_name(name, f'{name}_{col_name}_seq', used_index_names)
                    out.write(f'CREATE SEQUENCE {seq};\n')
                    table_sequences.append((seq, col_name))
                    extra = f' DEFAULT nextval({literal(seq)}::regclass)'
                else:
                    extra = ' GENERATED BY DEFAULT AS IDENTITY' if auto else ''
                if re.search(r'(?i)\bDEFAULT\b', decl) and not default:
                    raise ConversionError(f'Unsupported DEFAULT expression in {name}.{col_name}')
                if default and default.group(1).upper() != 'NULL':
                    if auto:
                        raise ConversionError(f'AUTO_INCREMENT has explicit DEFAULT in {name}.{col_name}')
                    extra += ' DEFAULT ' + pg_value(default.group(1), coltype, boolean)
                if null:
                    extra += ' NOT NULL'
                if inline_pk:
                    if primary:
                        raise ConversionError(f'Multiple primary keys in {name}')
                    primary = [ident(col_name)]
                lines.append(f'{ident(col_name)} {coltype}{extra}')
                cols.append((col_name, coltype, boolean))
                if auto:
                    sequences.append((name, col_name))
            if not cols:
                raise ConversionError(f'Table has no supported columns: {name}')
            if primary:
                lines.append('PRIMARY KEY (' + ', '.join(primary) + ')')
            out.write(f'CREATE TABLE {ident(name)} (\n    ' + ',\n    '.join(lines) + '\n);\n')
            for seq, col_name in table_sequences:
                out.write(f'ALTER SEQUENCE {seq} OWNED BY {ident(name)}.{ident(col_name)};\n')
            out.write('\n')
            tables[name] = cols
            for unique, keyname, keycols in pending:
                idxname = index_name(name, keyname, used_index_names)
                indexes.append(f'CREATE {"UNIQUE " if unique else ""}INDEX {idxname} ON {ident(name)} ({", ".join(keycols)});')
            continue
        insert = re.match(r'(?is)^INSERT\s+INTO\s+(' + MYSQL_NAME + r')\s*(.*)$', stmt)
        if insert:
            name = mysql_name(insert.group(1))
            if name not in tables:
                raise ConversionError(f'INSERT references missing table: {name}')
            remainder = insert.group(4).strip()
            if remainder.startswith('('):
                colspec, remainder = take_parenthesized(remainder)
                colnames = [mysql_name(x) for x in split_top(colspec)]
            else:
                colnames = [x[0] for x in tables[name]]
            if len(set(colnames)) != len(colnames):
                raise ConversionError(f'Duplicate INSERT columns in {name}')
            types = {x[0]: (x[1], x[2]) for x in tables[name]}
            if any(x not in types for x in colnames):
                raise ConversionError(f'INSERT has an unknown column in {name}')
            val_match = re.match(r'(?is)^VALUES\s*(.*)$', remainder)
            if not val_match:
                raise ConversionError(f'Unsupported INSERT syntax in {name}')
            tuples = split_top(val_match.group(1))
            if not tuples:
                raise ConversionError(f'INSERT contains no rows in {name}')
            prepared = []
            for row in tuples:
                values, rest = take_parenthesized(row)
                if rest:
                    raise ConversionError(f'Unsupported INSERT suffix in {name}: {rest[:80]}')
                data = split_top(values)
                if len(data) != len(colnames):
                    raise ConversionError(f'Column count differs in {name}')
                prepared.append('(' + ', '.join(pg_value(token, *types[col]) for col, token in zip(colnames, data)) + ')')
                rows_count += 1
            out.write(f'INSERT INTO {ident(name)} ({", ".join(ident(x) for x in colnames)}) VALUES\n' + ',\n'.join(prepared) + ';\n')
            continue
        if re.match(r'(?is)^(?:ALTER\s+TABLE|CREATE\s+INDEX|CREATE\s+UNIQUE\s+INDEX|REPLACE\s+INTO|UPDATE\s+|DELETE\s+|TRUNCATE\s+|CREATE\s+(?:VIEW|TRIGGER|PROCEDURE|FUNCTION)|INSERT\s+)', stmt):
            raise ConversionError(f'Unsupported MySQL statement: {stmt[:150]!r}')
        raise ConversionError(f'Unknown MySQL statement: {stmt[:150]!r}')
    if not tables:
        raise ConversionError('No CREATE TABLE statements found in the dump')
    if indexes:
        out.write('\n' + '\n'.join(indexes) + '\n')
    if fks:
        out.write('\n' + '\n'.join(fks) + '\n')
    emit_sequences(out, sequences)
    out.write('\nCOMMIT;\n')
    return len(tables), rows_count


def emit_sequences(out: TextIO, sequences: list[tuple[str, str]]) -> None:
    for table, col in sequences:
        # pg_get_serial_sequence works for both serial and identity columns.
        out.write(f'SELECT setval(pg_get_serial_sequence({literal(ident(table))}, {literal(col)}), '
                  f'GREATEST(COALESCE((SELECT MAX({ident(col)})::bigint FROM {ident(table)}), 1), 1), '
                  f'(SELECT COUNT(*) > 0 FROM {ident(table)}));\n')


def sqlite_type(original: str) -> tuple[str, bool]:
    t = original.upper().strip()
    if 'BOOL' in t:
        return 'BOOLEAN', True
    if re.match(r'^(?:DATETIME|TIMESTAMP)\b', t):
        return 'TIMESTAMP', False
    if re.match(r'^DATE\b', t):
        return 'DATE', False
    if re.match(r'^TIME\b', t):
        return 'TIME', False
    if 'INT' in t:
        return 'BIGINT', False
    if 'CHAR' in t or 'CLOB' in t or 'TEXT' in t or not t:
        return 'TEXT', False
    if 'BLOB' in t:
        return 'BYTEA', False
    if 'REAL' in t or 'FLOA' in t or 'DOUB' in t:
        return 'DOUBLE PRECISION', False
    if 'DECIMAL' in t or 'NUMERIC' in t:
        match = re.match(r'(?i)^(?:DECIMAL|NUMERIC)\s*(\(\d+(?:,\s*\d+)?\))?', t)
        return 'NUMERIC' + ((match.group(1).replace(' ', '')) if match and match.group(1) else ''), False
    if 'JSON' in t:
        return 'JSONB', False
    raise ConversionError(f'Unsupported SQLite type {original!r}')



def sqlite_default(value: str, pgtype: str, boolean: bool) -> str:
    """SQLite string defaults do not use MySQL backslash escapes."""
    token = value.strip()
    if len(token) >= 2 and token[0] in ("'", '"') and token[-1] == token[0]:
        quote = token[0]
        decoded = token[1:-1].replace(quote + quote, quote)
        if boolean:
            if decoded not in ('0', '1'):
                raise ConversionError(f'Invalid SQLite boolean default {decoded!r}')
            return 'TRUE' if decoded == '1' else 'FALSE'
        if pgtype == 'BYTEA':
            return f"decode('{decoded.encode('utf-8').hex()}', 'hex')"
        if pgtype.startswith(('DATE', 'TIMESTAMP')) and decoded.startswith('0000-00-00'):
            raise ConversionError('Zero date cannot be represented in PostgreSQL')
        return literal(decoded)
    return pg_value(token, pgtype, boolean)


def sqlite_value(value, pgtype: str, boolean: bool) -> str:
    if value is None:
        return 'NULL'
    if isinstance(value, (bytes, bytearray, memoryview)):
        if pgtype != 'BYTEA':
            raise ConversionError('SQLite binary data found outside BYTEA column')
        return f"decode('{bytes(value).hex()}', 'hex')"
    if boolean:
        if value not in (0, 1, False, True):
            raise ConversionError(f'Unexpected SQLite boolean value: {value!r}')
        return 'TRUE' if value else 'FALSE'
    if isinstance(value, str):
        if '\x00' in value:
            raise ConversionError('NUL in SQLite text value')
        if pgtype.startswith(('TIMESTAMP', 'DATE')) and value.startswith('0000-00-00'):
            raise ConversionError('Zero date cannot be represented in PostgreSQL')
        return literal(value)
    if isinstance(value, bool):
        return 'TRUE' if value else 'FALSE'
    if isinstance(value, (int, float)):
        if isinstance(value, float) and (value != value or value in (float('inf'), float('-inf'))):
            raise ConversionError('Nonfinite SQLite float value')
        return str(value)
    raise ConversionError(f'Unknown SQLite value type: {type(value)}')


def convert_sqlite(source: Path, out: TextIO, batch_size: int = 250) -> tuple[int, int]:
    db = sqlite3.connect(source.resolve().as_uri() + '?mode=ro', uri=True)
    try:
        db.execute('PRAGMA query_only = ON')
        db.execute('BEGIN')
        entries = db.execute("SELECT name, type, sql FROM sqlite_master WHERE name NOT LIKE 'sqlite_%' ORDER BY type, name").fetchall()
        unsupported = [(name, kind) for name, kind, _ in entries if kind not in ('table', 'index')]
        if unsupported:
            raise ConversionError(f'Unsupported SQLite object: {unsupported[0]}')
        tables = [(name, ddl) for name, kind, ddl in entries if kind == 'table']
        if not tables:
            raise ConversionError('No SQLite tables found')
        indexes: list[str] = []
        fks: list[str] = []
        sequences: list[tuple[str, str]] = []
        used_indexes: set[str] = set()
        types: dict[str, list[tuple[str, str, bool]]] = {}
        out.write('BEGIN;\nSET client_encoding = \'UTF8\';\nSET standard_conforming_strings = on;\n\n')
        for table, ddl in tables:
            if re.search(r'\bCHECK\s*\(|\bCOLLATE\s+NOCASE\b', ddl or '', re.I):
                raise ConversionError(f'Unsupported CHECK or NOCASE collation in {table}')
            escaped = '"' + table.replace('"', '""') + '"'
            columns = db.execute(f'PRAGMA table_xinfo({escaped})').fetchall()
            if any(c[6] for c in columns):
                raise ConversionError(f'Generated or hidden SQLite column in {table}')
            if not columns:
                raise ConversionError(f'No columns found in {table}')
            pk = sorted((int(c[5]), c[1]) for c in columns if c[5])
            parts = []
            type_entries = []
            for _, name, dtype, required, default, pk_idx, _ in columns:
                ptype, boolean = sqlite_type(dtype or '')
                identity = bool(len(pk) == 1 and pk_idx == 1 and (dtype or '').upper().strip() == 'INTEGER' and 'WITHOUT ROWID' not in (ddl or '').upper())
                extra = ' GENERATED BY DEFAULT AS IDENTITY' if identity else ''
                if default is not None and str(default).upper() != 'NULL':
                    if identity:
                        raise ConversionError(f'Identity column has default in {table}.{name}')
                    extra += ' DEFAULT ' + sqlite_default(str(default), ptype, boolean)
                if required or pk_idx:
                    extra += ' NOT NULL'
                parts.append(f'{ident(name)} {ptype}{extra}')
                type_entries.append((name, ptype, boolean))
                if identity:
                    sequences.append((table, name))
            if pk:
                parts.append('PRIMARY KEY (' + ', '.join(ident(col) for _, col in pk) + ')')
            out.write(f'CREATE TABLE {ident(table)} (\n    ' + ',\n    '.join(parts) + '\n);\n\n')
            types[table] = type_entries
            for seq, idx_name, unique, origin, partial in db.execute(f'PRAGMA index_list({escaped})').fetchall():
                if partial:
                    raise ConversionError(f'Partial SQLite index unsupported: {table}.{idx_name}')
                quoted_idx = '"' + idx_name.replace('"', '""') + '"'
                cols = db.execute(f'PRAGMA index_info({quoted_idx})').fetchall()
                if not cols or any(c[2] is None for c in cols):
                    raise ConversionError(f'Expression SQLite index unsupported: {table}.{idx_name}')
                if origin == 'pk':
                    continue
                new_name = index_name(table, idx_name if not idx_name.startswith('sqlite_autoindex_') else 'uidx_' + table + '_' + str(seq), used_indexes)
                indexes.append(f'CREATE {"UNIQUE " if unique else ""}INDEX {new_name} ON {ident(table)} ({", ".join(ident(c[2]) for c in cols)});')
            groups: dict[int, list[tuple]] = defaultdict(list)
            for row in db.execute(f'PRAGMA foreign_key_list({escaped})'):
                groups[row[0]].append(row)
            for group in groups.values():
                group.sort(key=lambda x: x[1])
                ref = group[0][2]
                delete_action = group[0][6].upper()
                update_action = group[0][5].upper()
                fks.append(f'ALTER TABLE {ident(table)} ADD FOREIGN KEY (' + ', '.join(ident(x[3]) for x in group) + f') REFERENCES {ident(ref)} (' + ', '.join(ident(x[4]) for x in group) + f') ON UPDATE {update_action} ON DELETE {delete_action};')
        rowcount = 0
        for table, cols in types.items():
            projection = ', '.join(ident(c[0]) for c in cols)
            cur = db.execute(f'SELECT {projection} FROM {ident(table)}')
            while rows := cur.fetchmany(batch_size):
                converted = ["(" + ', '.join(sqlite_value(value, pgtype, boolean) for value, (_, pgtype, boolean) in zip(row, cols)) + ")" for row in rows]
                out.write(f'INSERT INTO {ident(table)} ({projection}) VALUES\n' + ',\n'.join(converted) + ';\n')
                rowcount += len(rows)
        if indexes:
            out.write('\n' + '\n'.join(indexes) + '\n')
        if fks:
            out.write('\n' + '\n'.join(fks) + '\n')
        emit_sequences(out, sequences)
        out.write('\nCOMMIT;\n')
        db.execute('ROLLBACK')
        return len(tables), rowcount
    finally:
        db.close()


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--source', choices=('mysql', 'sqlite'), required=True)
    parser.add_argument('--input', required=True, type=Path, help='MySQL dump (.sql) or SQLite database (.db)')
    parser.add_argument('--output', required=True, type=Path, help='Destination SQL file; never uploaded')
    parser.add_argument('--force', action='store_true', help='Overwrite an existing output file')
    parser.add_argument('--batch-size', type=int, default=250, help='SQLite INSERT rows per statement')
    args = parser.parse_args(argv)
    if not args.input.is_file():
        parser.error('Input file does not exist')
    if args.batch_size < 1 or args.batch_size > 5000:
        parser.error('--batch-size must be between 1 and 5000')
    if args.output.exists() and not args.force:
        parser.error('Output exists; use --force to overwrite it')
    if args.input.resolve() == args.output.resolve():
        parser.error('Input and output must be different files')
    fd, temp_path = tempfile.mkstemp(prefix='.postgres-convert-', suffix='.sql', dir=args.output.parent)
    try:
        with os.fdopen(fd, 'w', encoding='utf-8', newline='\n') as out:
            if args.source == 'mysql':
                with args.input.open('r', encoding='utf-8-sig') as stream:
                    table_count, row_count = convert_mysql(stream, out)
            else:
                table_count, row_count = convert_sqlite(args.input, out, args.batch_size)
        if args.output.exists() and not args.force:
            raise ConversionError('Output was created by another process')
        os.replace(temp_path, args.output)
        print(f'Converted {table_count} tables and {row_count} rows to {args.output}', file=sys.stderr)
        return 0
    except (ConversionError, OSError, UnicodeError, sqlite3.Error) as exc:
        print(f'Conversion failed: {exc}', file=sys.stderr)
        return 1
    finally:
        if os.path.exists(temp_path):
            os.unlink(temp_path)


if __name__ == '__main__':
    sys.exit(main())