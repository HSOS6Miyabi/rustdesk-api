#!/usr/bin/env python3
"""Check the HTTP contracts used by the browser clients against a fresh API database."""

import contextlib
import json
from pathlib import Path
import re
import secrets
import shutil
import signal
import socket
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.parse
import urllib.request

ROOT = Path(__file__).resolve().parents[1]
PREFIX = "/api/admin"


def check(value, message):
    if not value:
        raise AssertionError(message)


def backend_routes():
    routes = set()
    func = re.compile(r"(?m)^func\s+(\w+)\s*\(([^)]*)\)\s*\{")
    group = re.compile(r"(\w+)\s*:?=\s*(\w+)\.Group\(\"([^\"]+)\"\)")
    alias = re.compile(r"(\w+)\s*:?=\s*(\w+)\.Use\(")
    method = re.compile(r'\b(\w+)\.(GET|POST|PUT|PATCH|DELETE|OPTIONS)\("([^"]+)"')
    for path in (ROOT / "http/router/admin.go", ROOT / "http/controller/admin/device_deployment.go"):
        source = path.read_text(encoding="utf-8")
        for item in func.finditer(source):
            name = item.group(1)
            if not (name.endswith("Bind") or name == "Init" and path.name == "admin.go"):
                continue
            depth = 1
            end = item.end()
            while depth and end < len(source):
                depth += (source[end] == "{") - (source[end] == "}")
                end += 1
            variables = {item.group(2).split()[0]: "" if name == "Init" else PREFIX}
            for line in source[item.end():end-1].splitlines():
                for match in group.finditer(line):
                    child, parent, suffix = match.groups()
                    if parent in variables:
                        variables[child] = variables[parent] + suffix
                for match in alias.finditer(line):
                    child, parent = match.groups()
                    if parent in variables:
                        variables[child] = variables[parent]
                for match in method.finditer(line):
                    parent, verb, suffix = match.groups()
                    if parent in variables:
                        routes.add((verb, variables[parent] + suffix))
    return routes


def browser_routes():
    routes = set()
    pages = {}
    for path in (ROOT / "resources/admin/app.js", ROOT / "resources/web2/app.js"):
        code = path.read_text(encoding="utf-8")
        for match in re.finditer(
            r'''request\(\s*["'](/[^"']+)["']\s*(?:,\s*["'](GET|POST|PATCH|PUT|DELETE)["'])?''', code
        ):
            uri, method = match.groups()
            routes.add((method or "GET", PREFIX + uri.split("?")[0]))
        for match in re.finditer(
            r'''(?m)^\s*(\w+):\s*\{[^\n]*?endpoint:"([^"]+)"''', code
        ):
            pages[match.group(1)] = match.group(2)
    readonly = {"deployed", "trash", "myconnections", "mytransfers", "myalarms",
                "alarms", "operations", "myloginlogs", "loginlogs", "connections",
                "transfers", "notifications", "activeSessions", "disconnectHistory", "sessions"}
    for page, endpoint in pages.items():
        uri = PREFIX + "/" + endpoint
        routes.add(("GET", uri + "/list"))
        if page not in readonly:
            if page != "mydevices":
                routes.add(("POST", uri + "/create"))
            if page not in ("invitations", "apiTokens"):
                routes.add(("POST", uri + "/update"))
                routes.add(("POST", uri + "/delete"))
    # The legacy WebClient sends two additional API requests outside the
    # administration URL prefix. Keep their HTTP methods in the inventory.
    legacy=(ROOT / "resources/web/js/src/ljw.js").read_text(encoding="utf-8")
    for match in re.finditer(r'fetch\(apiserver\s*\+\s*"(/api/[^"]+)"\s*,\s*\{\s*method:\s*["\'](\w+)["\']', legacy):
        routes.add((match.group(2).upper(), match.group(1)))
    for endpoint in ("address_book", "address_book_collection", "tag"):
        for operation in ("list", "create", "update", "delete"):
            routes.add(("GET" if operation == "list" else "POST",
                        PREFIX + "/my/" + endpoint + "/" + operation))
    extra = {
        "GET": ("/audit/export", "/client_builder/tasks/:id/download", "/my/group/members",
                "/my/group/anchors", "/group/members", "/group/anchors",
                "/address_book/export", "/my/address_book/export", "/my/peer/list"),
        "POST": ("/my/group/members/batch", "/my/group/anchors", "/group/members/batch",
                 "/group/anchors", "/address_book/import", "/my/address_book/import",
                 "/my/address_book/trash/batchRestore", "/my/address_book/trash/batchPurge",
                 "/my/address_book/trash/restore", "/my/address_book/trash/purge",
                 "/address_book/trash/batchRestore", "/address_book/trash/batchPurge",
                 "/address_book/trash/restore", "/address_book/trash/purge",
                 "/settings/overrides/reset", "/mfa/disable", "/mfa/recovery/regenerate",
                 "/my/peer/update", "/my/peer/delete", "/peer/transfer"),
        "PATCH": ("/settings/overrides",),
    }
    for verb, paths in extra.items():
        routes.update((verb, PREFIX + path) for path in paths)
    routes = {x for x in routes if not x[1].endswith("/client_builder/tasks/")}
    return routes, pages


class Client:
    def __init__(self, url, token=""):
        self.url, self.token, self.count = url, token, 0

    def send(self, method, path, body=None, raw=False):
        uri = path if path.startswith("/api/") else PREFIX + path
        headers = {"Accept": "application/json", "Accept-Language": "zh-CN"}
        if self.token:
            headers["api-token"] = self.token
        if body is not None:
            headers["Content-Type"] = "application/json"
        request = urllib.request.Request(
            self.url + uri, method=method, headers=headers,
            data=json.dumps(body).encode() if body is not None else None
        )
        try:
            with urllib.request.urlopen(request, timeout=7) as reply:
                status, payload = reply.status, reply.read(2_000_000)
                content_type = reply.headers.get("Content-Type", "")
        except urllib.error.HTTPError as error:
            status, payload = error.code, error.read(2_000_000)
            content_type = error.headers.get("Content-Type", "")
        self.count += 1
        check(status < 500, f"{method} {uri} returned HTTP {status}")
        if raw:
            return status, payload
        check(status != 404 or payload.startswith(b"{"), f"Unregistered API: {method} {uri}")
        check("json" in content_type or payload.startswith(b"{"),
              f"Unexpected response type for {method} {uri}: HTTP {status}")
        try:
            return status, json.loads(payload)
        except (ValueError, UnicodeDecodeError):
            raise AssertionError(f"Invalid JSON from {method} {uri}")

    def success(self, method, path, body=None):
        status, value = self.send(method, path, body)
        check(status == 200 and value.get("code") == 0,
              f"{method} {path}: HTTP {status}, {value.get('message', value.get('error'))}")
        return value.get("data")


@contextlib.contextmanager
def live_server(binary, directory, port):
    logfile = open(directory / "api-test.log", "ab", buffering=0)
    process = subprocess.Popen([str(binary)], cwd=directory, stdout=logfile,
                               stderr=subprocess.STDOUT, start_new_session=True)
    url = f"http://127.0.0.1:{port}"
    try:
        for _ in range(120):
            if process.poll() is not None:
                raise AssertionError("The test API exited during startup")
            try:
                with urllib.request.urlopen(url + PREFIX + "/config/admin", timeout=.5) as reply:
                    if reply.status == 200:
                        break
            except (urllib.error.URLError, TimeoutError):
                time.sleep(.1)
        else:
            raise AssertionError("The test API failed to start")
        yield Client(url)
    finally:
        if process.poll() is None:
            process.send_signal(signal.SIGTERM)
            try:
                process.wait(timeout=12)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait(timeout=5)
        logfile.close()


def install(directory):
    shutil.copytree(ROOT / "conf", directory / "conf")
    (directory / "data").mkdir()
    (directory / "runtime").mkdir()
    (directory / "resources").symlink_to(ROOT / "resources", target_is_directory=True)
    sock = socket.socket()
    sock.bind(("127.0.0.1", 0))
    port = sock.getsockname()[1]
    sock.close()
    path = directory / "conf/config.yaml"
    source = path.read_text()
    check('"0.0.0.0:21114"' in source, "Unexpected listener configuration")
    path.write_text(source.replace('"0.0.0.0:21114"', f'"127.0.0.1:{port}"'))
    return port


def all_route_requests(client, contracts):
    tested = set()
    invalid_responses = []
    for verb, route in sorted(contracts):
        if route.endswith("/logout"):
            continue
        if route in ("/api/shared-peer", "/api/server-config"):
            continue
        path = route.replace(":id", "0")
        if verb == "GET" and (path.endswith("/export") or path.endswith("/download")):
            client.send(verb, path, raw=True)
            tested.add((verb, route))
            continue
        if verb == "GET":
            params = {"page": "1", "page_size": "5"}
            if path.endswith("/members") or path.endswith("/anchors"):
                params["group_id"] = "1"
            if path.endswith("/lookup"):
                params["username"] = "no-such-user"
            path += "?" + urllib.parse.urlencode(params)
            status, result = client.send(verb, path)
            if route.endswith("/list") and status == 200 and result.get("code") == 0:
                value = result.get("data")
                is_list = isinstance(value, dict) and "list" in value and (
                    value["list"] is None or isinstance(value["list"], list))
                is_contacts = (route.endswith("/contact/list") and isinstance(value, dict)
                               and "incoming" in value and "outgoing" in value)
                if not (is_list or is_contacts):
                    invalid_responses.append((route, str(value)[:200]))
        else:
            client.send(verb, path, {})
        tested.add((verb, route))
    check(not invalid_responses, f"Browser list response mismatches: {invalid_responses}")
    check(len(tested) >= 100, "Too few real browser API calls were tested")
    return len(tested)



def verify_browser_response_shapes(client):
    """Check properties actually accessed by the administration and WebClient screens."""
    contracts = {
        "/config/admin": ("title",),
        "/config/server": ("id_server", "relay_server", "api_server", "key"),
        "/login-options": ("ops", "need_captcha", "disable_pwd"),
        "/contact/list": ("incoming", "outgoing"),
        "/shared_devices/list?page=1&page_size=5": ("list",),
        "/shared_devices/sources": ("sources",),
        "/push/channels": ("list",),
        "/push/history?page=1&page_size=5": ("list", "total"),
        "/notification/settings": ("settings", "catalog"),
        "/notification/list?page=1&page_size=5": ("list", "total"),
        "/mail/settings": ("enabled",),
        "/client_builder/profiles?page=1&page_size=5": ("list",),
        "/client_builder/tasks?page=1&page_size=5": ("list",),
        "/settings/effective": ("items",),
        "/settings/overview": ("summary", "outbox"),
        "/settings/history?limit=5": ("list",),
        "/settings/outbox?category=push": ("list",),
        "/mfa/status": ("enabled",),
        "/mfa/email/status": ("enabled",),
    }
    for path, keys in contracts.items():
        data=client.success("GET",path)
        check(isinstance(data,dict), f"Invalid browser response object: {path}")
        missing=set(keys)-set(data)
        check(not missing, f"Browser response at {path} is missing: {sorted(missing)}")
    device_data=client.success("POST","/geoip/lookup",{
        "ips":["127.0.0.1"],"device_ids":["does-not-exist"]
    })
    check(isinstance(device_data,dict)
          and "locations" in device_data and "device_status" in device_data,
          "Device presence response is incompatible with the browser")


def check_real_changes(admin):
    username = "contract_" + secrets.token_hex(4)
    password = "Contract-" + secrets.token_hex(8)
    admin.success("POST", "/user/create", {
        "username": username, "password": password, "nickname": "First",
        "group_id": 1, "is_admin": False, "status": 1, "remark": "",
    })
    users = admin.success("GET", "/user/list?keyword=" + username + "&page=1&page_size=30")
    user_id = next(row["id"] for row in users["list"] if row["username"] == username)
    admin.success("POST", "/user/update", {
        "id": user_id, "username": username, "nickname": "Updated",
        "group_id": 1, "is_admin": False, "status": 1, "remark": "",
    })
    check(admin.success("GET", f"/user/detail/{user_id}")["nickname"] == "Updated",
          "Account editing did not persist")
    for resource, name in (("/group", "Global group"), ("/my/group", "Personal group")):
        admin.success("POST", resource + "/create", {"name": name, "type": 1})
        rows = admin.success("GET", resource + "/list?page=1&page_size=100")["list"]
        group_id = next(row["id"] for row in rows if row["name"] == name)
        admin.success("POST", resource + "/update", {"id": group_id, "name": name + " updated", "type": 1})
        check(any(row["name"] == name + " updated" for row in
                  admin.success("GET", resource + "/list?page=1&page_size=100")["list"]),
              "Group changes did not persist")
        admin.success("POST", resource + "/delete", {"id": group_id})
    device_id = "contract-device-" + secrets.token_hex(4)
    admin.success("POST", "/peer/create", {"id": device_id, "hostname": "Test device", "alias": "First"})
    peers = admin.success("GET", "/peer/list?keyword=" + device_id + "&page=1&page_size=30")
    row_id = next(row["row_id"] for row in peers["list"] if row["id"] == device_id)
    admin.success("POST", "/peer/transfer", {"row_id": row_id, "user_id": user_id})
    user = Client(admin.url)
    response = user.success("POST", "/login", {"username": username, "password": password, "platform": "web"})
    user.token = response["token"]
    rows = user.success("GET", "/my/peer/list?keyword=" + device_id + "&page=1&page_size=30")["list"]
    check(any(row["id"] == device_id for row in rows), "Device is missing from owner listing")
    user.success("POST", "/my/peer/update", {"row_id": row_id, "alias": "Personal update", "note": "Owner note"})
    rows = user.success("GET", "/my/peer/list?keyword=" + device_id + "&page=1&page_size=30")["list"]
    check(any(row["alias"] == "Personal update" and row["note"] == "Owner note" for row in rows),
          "Personal device editing did not persist")
    collection_name = "Contract collection " + secrets.token_hex(3)
    user.success("POST", "/my/address_book_collection/create", {"name": collection_name})
    collections = user.success("GET", "/my/address_book_collection/list?page=1&page_size=100")["list"]
    collection_id = next(row["id"] for row in collections if row["name"] == collection_name)
    user.success("POST", "/my/address_book_collection/update", {
        "id": collection_id, "name": "Renamed collection", "user_id": 999999,
    })
    collections = user.success("GET", "/my/address_book_collection/list?page=1&page_size=100")["list"]
    changed = next(row for row in collections if row["id"] == collection_id)
    check(changed["name"] == "Renamed collection" and changed["user_id"] == user_id,
          "A personal collection changed its owner or failed to update")

    entry_id = "book-" + secrets.token_hex(4)
    user.success("POST", "/my/address_book/create", {
        "id": entry_id, "alias": "Original alias", "hostname": "Original host",
        "collection_id": collection_id, "hash": "keep-original-hash", "tags": ["keep-tag"],
    })
    address_books = user.success("GET", "/my/address_book/list?page=1&page_size=100")["list"]
    entry = next(row for row in address_books if row["id"] == entry_id)
    user.success("POST", "/my/address_book/update", {
        "row_id": entry["row_id"], "id": entry_id, "alias": "Updated alias",
        "hostname": "Updated host", "collection_id": collection_id,
    })
    address_books = user.success("GET", "/my/address_book/list?page=1&page_size=100")["list"]
    entry = next(row for row in address_books if row["id"] == entry_id)
    check(entry["alias"] == "Updated alias" and entry["hash"] == "keep-original-hash"
          and entry["user_id"] == user_id and "keep-tag" in json.dumps(entry.get("tags")),
          "Personal address book edits erased fields or changed the owner")

    user.success("POST", "/my/address_book_collection_rule/create", {
        "collection_id": collection_id, "type": 3, "to_id": 0, "rule": 1,
    })
    rules = user.success("GET", "/my/address_book_collection_rule/list?page=1&page_size=100")["list"]
    rule_id = next(row["id"] for row in rules if row["collection_id"] == collection_id)
    user.success("POST", "/my/address_book_collection_rule/update", {
        "id": rule_id, "collection_id": collection_id, "type": 3, "to_id": 0,
        "rule": 2, "user_id": 999999,
    })
    rules = user.success("GET", "/my/address_book_collection_rule/list?page=1&page_size=100")["list"]
    check(any(row["id"] == rule_id and row["rule"] == 2 and row["user_id"] == user_id for row in rules),
          "Personal address book rule edit failed or reassigned its owner")

    tag_name = "Contract label " + secrets.token_hex(3)
    user.success("POST", "/my/tag/create", {"name": tag_name, "color": 4282744575,
                                              "collection_id": collection_id})
    tags = user.success("GET", "/my/tag/list?page=1&page_size=100")["list"]
    tag_id = next(row["id"] for row in tags if row["name"] == tag_name)
    user.success("POST", "/my/tag/update", {
        "id": tag_id, "name": "Updated label", "color": 4282744575,
        "collection_id": collection_id,
    })
    tags = user.success("GET", "/my/tag/list?page=1&page_size=100")["list"]
    check(any(row["id"] == tag_id and row["name"] == "Updated label"
              and row["user_id"] == user_id for row in tags),
          "Personal tag editing failed or changed its owner")
    user.success("POST", "/my/tag/update", {
        "id": tag_id, "name": "Transparent label", "color": 0,
        "collection_id": collection_id,
    })
    tags = user.success("GET", "/my/tag/list?page=1&page_size=100")["list"]
    check(any(row["id"] == tag_id and row["color"] == 0 for row in tags),
          "A transparent black ARGB color must be saved without modification")

    _, denied = user.send("POST", "/user/create", {"username": "denied", "password": "denied"})
    check(denied.get("code") != 0, "An ordinary user could manage administrator accounts")
    other_name = "contract_other_" + secrets.token_hex(3)
    other_password = "Other-" + secrets.token_hex(8)
    admin.success("POST", "/user/create", {
        "username": other_name, "password": other_password, "group_id": 1,
        "is_admin": False, "status": 1,
    })
    stranger = Client(admin.url)
    stranger.token = stranger.success("POST", "/login", {
        "username": other_name, "password": other_password, "platform": "web",
    })["token"]
    _, denied = stranger.send("POST", "/my/address_book/update", {
        "row_id": entry["row_id"], "id": entry_id, "alias": "Unauthorized edit",
    })
    check(denied.get("code") != 0, "Another account can modify a private address book")
    _, denied = stranger.send("POST", "/my/address_book_collection/update", {
        "id": collection_id, "name": "Unauthorized rename",
    })
    check(denied.get("code") != 0, "Another account can modify a private collection")
    for path, payload in (
        ("/my/peer/update",{"row_id":row_id,"alias":"Unauthorized"}),
        ("/my/tag/update",{"id":tag_id,"name":"Unauthorized","color":0}),
        ("/my/address_book_collection_rule/update",{
            "id":rule_id,"collection_id":collection_id,"type":3,"to_id":0,"rule":2,
        }),
    ):
        _,denied=stranger.send("POST",path,payload)
        check(denied.get("code")!=0,f"Cross-account modification accepted at {path}")
    for path in (
        "/user/list","/peer/list","/group/list","/admin_role/list",
        "/settings/effective","/audit_conn/list","/device_deployment/list",
    ):
        status,denied=stranger.send("GET",path+"?page=1&page_size=5")
        check(status!=200 or denied.get("code")!=0,
              f"An ordinary user could read administrator-only records at {path}")
    for method,path,payload in (
        ("POST","/user/delete",{"id":user_id}),
        ("POST","/peer/delete",{"row_id":row_id}),
        ("POST","/group/update",{"id":1,"name":"Unauthorized","type":1}),
        ("PATCH","/settings/overrides",{"changes":[]}),
    ):
        _,denied=stranger.send(method,path,payload)
        check(denied.get("code")!=0,
              f"An ordinary user could change administrator-only data at {path}")
    # Verify searches, details and actual shared-device permissions from normal sessions.
    user_matches=user.success("GET","/my/directory/users?keyword="+urllib.parse.quote(username[:10]))["list"]
    check(any(record["id"]==user_id for record in user_matches),
          "Ordinary account directory cannot find a username")
    nickname_matches=user.success("GET","/my/directory/users?keyword=Updated")["list"]
    check(any(record["id"]==user_id for record in nickname_matches),
          "Ordinary account directory cannot search nicknames")
    detail=user.success("GET","/my/directory/user?id="+urllib.parse.quote(username))
    check(detail["id"]==user_id and "password" not in detail,
          "Public user details must be accessible without disclosing passwords")
    own=user.success("GET","/my/directory/device?id="+urllib.parse.quote(device_id))
    check(own["id"]==device_id and own["owned"] is True,
          "Owned device details do not match the logged-in owner")
    own_results=user.success("GET","/my/directory/devices?keyword=Personal")["list"]
    check(any(item["id"]==device_id for item in own_results),
          "Device search does not search aliases")
    _,unshared=stranger.send("GET","/my/directory/device?id="+urllib.parse.quote(device_id))
    check(unshared.get("code")!=0,"An unrelated user could read another user's device")
    _,unshared_search=stranger.send("GET","/my/directory/devices?keyword="+urllib.parse.quote(device_id))
    check(unshared_search.get("code")==0 and not unshared_search["data"]["list"],
          "An unrelated user could find another user's private devices")
    user.success("POST","/contact/add",{"username":other_name})
    shared=stranger.success("GET","/my/directory/device?id="+urllib.parse.quote(device_id))
    check(shared["id"]==device_id and shared["owned"] is False
          and shared["owner_username"]==username,
          "Shared device details were missing or had the wrong owner")
    shared_search=stranger.success("GET","/my/directory/devices?keyword="+urllib.parse.quote(device_id))["list"]
    check(any(row["id"]==device_id for row in shared_search),
          "Shared device search does not return permitted devices")
    stranger.success("GET","/my/directory/user?id="+urllib.parse.quote(username))
    user.success("POST","/contact/delete",{"username":other_name})
    _,unshared=stranger.send("GET","/my/directory/device?id="+urllib.parse.quote(device_id))
    check(unshared.get("code")!=0,"Revoked device sharing retained access")

    # Sort two or more pages in the database before applying pagination.
    prefix="zsort_"+secrets.token_hex(3)
    names=[prefix+"_zulu",prefix+"_alpha",prefix+"_echo",prefix+"_bravo",
           prefix+"_delta",prefix+"_charlie",prefix+"_foxtrot",prefix+"_golf"]
    for name in names: admin.success("POST","/group/create",{"name":name,"type":1})
    def sorted_page(path,column,direction,per_page=3):
        first=admin.success("GET",path+"?page=1&page_size="+str(per_page)+
                            "&sort_by="+column+"&sort_dir="+direction)
        all_rows=list(first["list"])
        for page_index in range(2,(first["total"]+per_page-1)//per_page+1):
            nxt=admin.success("GET",path+"?page="+str(page_index)+
                              "&page_size="+str(per_page)+"&sort_by="+column+
                              "&sort_dir="+direction)
            all_rows.extend(nxt["list"])
        check(len(all_rows)==first["total"],"Sorting skipped or duplicated records")
        return all_rows
    ascending=sorted_page("/group/list","name","asc")
    check([row["name"] for row in ascending]==sorted(
          [row["name"] for row in ascending],key=str.casefold),
          "Group ordering must be consistent across pages")
    descending=sorted_page("/group/list","name","desc")
    check([row["name"] for row in descending]==sorted(
          [row["name"] for row in descending],key=str.casefold,reverse=True),
          "Descending sorting must operate over all pages")
    users_asc=sorted_page("/user/list","username","asc",1)
    check([x["username"] for x in users_asc]==sorted(
          [x["username"] for x in users_asc],key=str.casefold),
          "User sorting did not apply before pagination")
    peers_asc=sorted_page("/peer/list","id","asc",1)
    check([x["id"] for x in peers_asc]==sorted(
          [x["id"] for x in peers_asc],key=str.casefold),
          "Device sorting did not apply before pagination")
    admin.success("GET","/user/list?page=1&page_size=5&sort_by="+
                  urllib.parse.quote("username;DROP TABLE users")+"&sort_dir=desc")
    check(admin.success("GET","/user/list?page=1&page_size=3")["total"]>=3,
          "An unsupported sort column affected stored accounts")
    return user.count + stranger.count


def main():
    browser, pages = browser_routes()
    backend = backend_routes()
    backend.update({("POST","/api/shared-peer"),("POST","/api/server-config"),("POST","/api/server-config-v2")})
    missing = browser - backend
    check(not missing, f"Unregistered browser API calls: {sorted(missing)}")
    check(len(browser) >= 212 and len(pages) >= 35, "The browser API inventory is incomplete")
    if len(sys.argv) == 2 and sys.argv[1] == "--static":
        print(f"Matched {len(browser)} browser routes, {len(pages)} views, {len(backend)} server routes")
        return
    check(len(sys.argv) == 2, "Provide an API executable or --static")
    binary = Path(sys.argv[1]).resolve()
    with tempfile.TemporaryDirectory(prefix="rustdesk-api-contract-") as temporary:
        directory = Path(temporary)
        port = install(directory)
        with live_server(binary, directory, port):
            pass
        password = "Admin-" + secrets.token_hex(9)
        reset = subprocess.run([str(binary), "reset-admin-pwd", password],
                               cwd=directory, stdout=subprocess.DEVNULL,
                               stderr=subprocess.DEVNULL, timeout=45)
        check(reset.returncode == 0, "Administrator password setup failed")
        with live_server(binary, directory, port) as client:
            result = client.success("POST", "/login", {"username": "admin", "password": password, "platform": "web"})
            client.token = result["token"]
            headers={"Authorization":"Bearer "+client.token,"Content-Type":"application/json"}
            webclient=urllib.request.Request(
                client.url+"/api/server-config", data=b"{}",
                headers=headers, method="POST"
            )
            with urllib.request.urlopen(webclient, timeout=7) as reply:
                config=json.load(reply)
            check(config.get("code")==0 and isinstance(config["data"].get("peers"),dict),
                  "Legacy WebClient server-config did not return its required data")
            status,share=client.send("POST","/api/shared-peer",{"share_token":123})
            check(status<500 and share.get("code")!=0,
                  "Malformed WebClient sharing token must be rejected safely")
            for invalid_share in ({}, {"share_token":None}, {"share_token":[]}, {"share_token":""}):
                status,result=client.send("POST","/api/shared-peer",invalid_share)
                check(status<500 and result.get("code")!=0,
                      "Invalid shared-peer payload should return a validation error")
            webclient_v2=urllib.request.Request(
                client.url+"/api/server-config-v2", data=b"{}",
                headers=headers, method="POST")
            with urllib.request.urlopen(webclient_v2, timeout=7) as reply:
                response=json.load(reply)
            check(response.get("code")==0 and "id_server" in response.get("data",{}),
                  "Legacy WebClient server-config-v2 is inconsistent")
            check("*" in (client.success("GET", "/user/current").get("route_names") or []),
                  "Administrator authentication failed")
            tested = all_route_requests(client, browser)
            verify_browser_response_shapes(client)
            personal_requests = check_real_changes(client)
            logout_client = Client(client.url)
            logout_client.token=logout_client.success("POST","/login",{
                "username":"admin","password":password,"platform":"web"
            })["token"]
            logout_client.success("POST","/logout",{})
            _,revoked=logout_client.send("GET","/user/current")
            check(revoked.get("code")!=0,"Logout did not revoke the browser session")
            print(f"Passed {tested} browser API routes, {client.count + personal_requests + logout_client.count + 7} real HTTP requests, "
                  f"{len(pages)} views, and authenticated administrator/personal CRUD operations.")


if __name__ == "__main__":
    main()
