#!/usr/bin/env python3
"""Validate the manifest embedded in the release EXE, not just its source XML."""
import argparse
import re
import struct
import xml.etree.ElementTree as ET
from pathlib import Path


def embedded_manifest(path):
    data = Path(path).read_bytes()
    if data[:2] != b'MZ':
        raise ValueError('not a Windows executable')
    pe = struct.unpack_from('<I', data, 0x3c)[0]
    if data[pe:pe + 4] != b'PE\0\0':
        raise ValueError('invalid PE signature')
    section_count = struct.unpack_from('<H', data, pe + 6)[0]
    optional_size = struct.unpack_from('<H', data, pe + 20)[0]
    optional = pe + 24
    magic = struct.unpack_from('<H', data, optional)[0]
    directory = optional + (112 if magic == 0x20b else 96)
    resource_rva = struct.unpack_from('<I', data, directory + 16)[0]
    sections = []
    for index in range(section_count):
        section = optional + optional_size + index * 40
        size, rva, raw_size, raw = struct.unpack_from('<IIII', data, section + 8)
        sections.append((rva, max(size, raw_size), raw))

    def offset(rva):
        for start, size, raw in sections:
            if start <= rva < start + size:
                return raw + rva - start
        raise ValueError('resource RVA outside executable sections')

    base = offset(resource_rva)

    def entries(relative):
        position = base + relative
        named, ids = struct.unpack_from('<HH', data, position + 12)
        return [struct.unpack_from('<II', data, position + 16 + i * 8)
                for i in range(named + ids)]

    kinds = {key for key, _ in entries(0)}
    if not {3, 14}.issubset(kinds):
        raise ValueError("missing embedded application icon resources")
    manifest = next((value for key, value in entries(0) if key == 24), None)
    if manifest is None:
        raise ValueError('missing RT_MANIFEST resource')
    value = manifest
    while value & 0x80000000:
        children = entries(value & 0x7fffffff)
        # Resource ID 1 is the process manifest consumed by the Windows loader.
        value = next((target for key, target in children if key == 1), children[0][1])
    rva, size = struct.unpack_from('<II', data, base + value)
    return data[offset(rva):offset(rva) + size]


def validate(path):
    xml = embedded_manifest(path)
    if b'{{' in xml or b'}}' in xml:
        raise ValueError('unrendered template in embedded process manifest')
    root = ET.fromstring(xml)
    ns = {'a': 'urn:schemas-microsoft-com:asm.v1'}
    identity = root.find('a:assemblyIdentity', ns)
    if identity is None:
        raise ValueError('missing assembly identity')
    version = identity.get('version', '')
    parts = version.split('.')
    if len(parts) != 4 or any(not p.isdecimal() or int(p) > 65535 for p in parts):
        raise ValueError('assembly version must have four numeric components')
    name = identity.get('name', '')
    if not re.fullmatch(r'[A-Za-z0-9_.-]+', name):
        raise ValueError('invalid assembly name')
    for dependency in root.findall('a:dependency/a:dependentAssembly/a:assemblyIdentity', ns):
        if dependency.get('name') != 'Microsoft.Windows.Common-Controls':
            raise ValueError('unexpected side-by-side dependency')
        if dependency.get('version') != '6.0.0.0':
            raise ValueError('invalid Common Controls dependency version')
    return name, version


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('exe')
    args = parser.parse_args()
    try:
        name, version = validate(args.exe)
    except (ValueError, ET.ParseError, struct.error) as error:
        parser.exit(1, f'Windows resource validation failed: {error}\n')
    print(f'Windows process manifest valid: {name} {version}')
