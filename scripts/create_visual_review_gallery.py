#!/usr/bin/env python3
"""Build readable contact sheets from every browser audit screenshot."""
import json
import math
import sys
from collections import defaultdict
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont

source = Path(sys.argv[1])
target = Path(sys.argv[2])
target.mkdir(parents=True, exist_ok=True)
report = json.loads((source / "layout-report.json").read_text(encoding="utf-8"))
groups = defaultdict(list)
for row in report:
    if row.get("file") and (source / row["file"]).exists():
        groups[row["role"]].append(row)

font_paths = [
    Path("/home/runner/.local/share/fonts/NotoSansCJKsc-Regular.otf"),
    Path("/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf"),
]
font_file = next((str(path) for path in font_paths if path.exists()), None)
def font(size):
    return ImageFont.truetype(font_file, size) if font_file else ImageFont.load_default()

small_font = font(15)
title_font = font(21)
columns, rows_per_sheet = 3, 3
thumb_width, thumb_height = 590, 370
card_width, card_height = 610, 420
padding = 24

summaries = []
for role, items in sorted(groups.items()):
    for start in range(0, len(items), columns * rows_per_sheet):
        chunk = items[start:start+columns*rows_per_sheet]
        width = columns*card_width + 2*padding
        height = rows_per_sheet*card_height + 78
        sheet=Image.new("RGB", (width, height), "#f3f6fa")
        draw=ImageDraw.Draw(sheet)
        draw.text((padding, 16), role+"  ·  Pages "+str(start+1)+"–"+str(start+len(chunk)),
                  font=title_font,fill="#12273f")
        for i, item in enumerate(chunk):
            col, row = i%columns, i//columns
            x, y = padding+col*card_width, 58+row*card_height
            with Image.open(source / item["file"]) as original:
                original=original.convert("RGB")
                original.thumbnail((thumb_width,thumb_height))
                card=Image.new("RGB",(thumb_width,thumb_height),"#ffffff")
                card.paste(original,((thumb_width-original.width)//2,0))
                sheet.paste(card,(x,y+26))
            title=str(item["view"])+": "+str(item.get("title",""))
            draw.text((x+5,y+1),title[:50],font=small_font,fill="#153555")
            if item.get("unusedRight",0)>0 or item.get("panelWhitespace"):
                note="unused "+str(item.get("unusedRight",0))+"px"
                if item.get("panelWhitespace"):
                    note+=" · panel "+str(item["panelWhitespace"][:2])
                draw.text((x+5,y+394),note[:75],font=small_font,fill="#a44d22")
        name=role+"-"+str(start//(columns*rows_per_sheet)+1).zfill(2)+".png"
        sheet.save(target / name,optimize=True)
        summaries.append({"role":role,"name":name,"pages":[a["view"] for a in chunk]})

(target/"contact-sheets.json").write_text(json.dumps(summaries,ensure_ascii=False,indent=2),encoding="utf-8")
(target/"layout-report.json").write_text(json.dumps(report,ensure_ascii=False,indent=2),encoding="utf-8")
print("Generated",len(summaries),"contact sheets for",len(report),"page screenshots.")
