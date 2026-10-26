#!/usr/bin/env python3
"""Update all One Browser version metadata; minor and patch digits use base 10."""
import argparse
import json
from pathlib import Path
import re

ROOT = Path(__file__).resolve().parents[1]

def next_version(value):
    if not re.fullmatch(r"[0-9]\.[0-9]\.[0-9]", value):
        raise ValueError("版本必须是三位 0–9 数字，例如 2.0.0")
    number = int(value.replace(".", "")) + 1
    if number > 999:
        raise ValueError("已达到 9.9.9，请先确定下一阶段版本规则")
    return ".".join(f"{number:03d}")

def save_json(path, value):
    path.write_text(json.dumps(value, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")

def set_version(version):
    if not re.fullmatch(r"[0-9]\.[0-9]\.[0-9]", version):
        raise ValueError("版本必须是三位 0–9 数字")
    for relative in ["frontend/package.json", "frontend/package-lock.json"]:
        path = ROOT / relative
        data = json.loads(path.read_text(encoding="utf-8"))
        data["version"] = version
        if "packages" in data:
            data["packages"][""]["version"] = version
        save_json(path, data)
    path = ROOT / "wails.json"
    data = json.loads(path.read_text(encoding="utf-8"))
    data["info"]["productVersion"] = version
    save_json(path, data)
    path = ROOT / "build/windows/versioninfo.json"
    data = json.loads(path.read_text(encoding="utf-8"))
    major, minor, patch = map(int, version.split("."))
    for name in ["FileVersion", "ProductVersion"]:
        data["FixedFileInfo"][name] = {"Major": major, "Minor": minor, "Patch": patch, "Build": 0}
        data["StringFileInfo"][name] = version
    save_json(path, data)
    path = ROOT / "build/windows/wails.exe.manifest"
    text = path.read_text(encoding="utf-8")
    text = re.sub(r'(name="com\.onebrowser\.app" version=")[^"]+', lambda m: m[1] + version + ".0", text)
    path.write_text(text, encoding="utf-8")
    path = ROOT / "frontend/public/one-browser/index.html"
    text = path.read_text(encoding="utf-8")
    text = re.sub(r'(当前版本 <span>)[^<]+', lambda m: m[1] + version, text)
    text = re.sub(r'(id="clientCurrentVersion">)[^<]+', lambda m: m[1] + version, text)
    text = re.sub(r'(\?v=)[^"\s]+', lambda m: m[1] + version, text)
    path.write_text(text, encoding="utf-8")
    path = ROOT / "frontend/src/OneBrowserShell.tsx"
    text = re.sub(r'(index\.html\?v=)[^"\s]+', lambda m: m[1] + version, path.read_text(encoding="utf-8"))
    path.write_text(text, encoding="utf-8")


    path = ROOT / "frontend/public/one-browser/app.js"
    text = re.sub(r"(clientCurrentVersion=')[^']+", lambda m: m[1] + version, path.read_text(encoding="utf-8"))
    path.write_text(text, encoding="utf-8")
    path = ROOT / "backend/onebrowser_client_update.go"
    text = re.sub(r'(return ")[0-9]+\.[0-9]+\.[0-9]+(")', lambda m: m[1] + version + m[2], path.read_text(encoding="utf-8"))
    path.write_text(text, encoding="utf-8")
    path = ROOT / "backend/onebrowser_kernel_families.go"
    text = re.sub(r'OneBrowser/[0-9]+\.[0-9]+\.[0-9]+', 'OneBrowser/' + version, path.read_text(encoding="utf-8"))
    path.write_text(text, encoding="utf-8")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--set", dest="version", help="explicit initial version")
    args = parser.parse_args()
    current = json.loads((ROOT / "wails.json").read_text(encoding="utf-8"))["info"]["productVersion"]
    version = args.version or next_version(current)
    set_version(version)
    print(version)
