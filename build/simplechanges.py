#!/usr/bin/env python3

import codecs
import hashlib
import logging
import os.path
import subprocess
import sys
from collections import OrderedDict
from contextlib import contextmanager
from datetime import datetime
from shutil import rmtree
from tempfile import mkdtemp


logger = logging.getLogger('simplechanges')


def main():
    logging.basicConfig(
        level=logging.DEBUG,
        format="%(levelname)-8s %(message)s",
    )
    deb = sys.argv[1]
    deb = os.path.realpath(deb)

    with tmpdir():
        logger.info("Extracting control file.")
        subprocess.check_call(['dpkg-deb', '--control', deb])
        with codecs.open('DEBIAN/control', 'r', 'utf-8') as fo:
            controls = dict(parse(fo))
        changes = generate_changes(
            controls,
            filename=os.path.basename(deb),
            filesize=os.path.getsize(deb),
            md5=hashfile(deb, 'md5'),
            sha1=hashfile(deb, 'sha1'),
            sha256=hashfile(deb, 'sha256'),
        )

    for payload in format(changes):
        sys.stdout.write(payload)
    logger.info(".changes generated for %s.", deb)

    sys.exit(0)


@contextmanager
def tmpdir():
    oldpwd = os.getcwd()
    tmpdir = mkdtemp()
    logger.debug("Working in %s.", tmpdir)
    os.chdir(tmpdir)
    try:
        yield tmpdir
    finally:
        os.chdir(oldpwd)
        if os.environ.get('CLEAN') in {'0', 'n'}:
            logger.debug("Not cleaning %s.", tmpdir)
            return

        logger.debug("Cleaning %s.", tmpdir)
        rmtree(tmpdir)


def format(dict_):
    for key, value in dict_.items():
        if isinstance(value, list):
            value = '\n'.join(value) + '\n'

        if '\n' in value:
            payload = "%s:\n" % (key,)
            for line in value.splitlines():
                if line:
                    payload += " %s\n" % (line)
                else:
                    payload += " .\n"
            yield payload
        else:
            yield '%s: %s\n' % (key, value)


def parse(fo):
    key = None
    value = None
    for line in fo:
        if key is not None:
            if line.startswith(' '):
                line = line[1:]
                if '.' == line[0]:
                    value += '\n'
                else:
                    value += line
            else:
                key = None

        if key is None:
            a, b = line.split(':', 1)
            if b:
                yield a, b.strip()
            else:
                key = a
                value = ''


def hashfile(path, algorithm):
    hasher = getattr(hashlib, algorithm)()
    with open(path, 'rb') as fo:
        for chunk in iter(lambda: fo.read(4096), b""):
            hasher.update(chunk)
    return hasher.hexdigest()


CHANGELOG_FMT = u"""\
%(Source)s (%(Version)s) %(Distribution)s; urgency=low

  * New upstream version.
"""


def generate_changes(controls, filename, filesize, md5, sha1, sha256):
    changes = OrderedDict([
        ('Format', '1.8'),
        ('Date', datetime.utcnow().strftime('%c +0000')),
        ('Source', controls['Package']),
        ('Binary', controls['Package']),
        ('Distribution', os.environ['CODENAME']),
        ('Urgency', 'Low'),
    ])
    changes.update({
        k: v for k, v in controls.items()
        if k in {'Architecture', 'Description', 'Maintainer', 'Version'}
    })
    changes['Changed-By'] = u"%s <%s>" % (
        os.environ['DEBFULLNAME'],
        os.environ['DEBEMAIL'],
    )
    changes['Changes'] = CHANGELOG_FMT % changes

    changes['Files'] = [u' '.join([
        md5,
        str(filesize),
        controls['Section'] or 'default',
        controls['Priority'],
        filename,
    ])]
    changes['Checksums-Sha1'] = [u' '.join([
        sha1, str(filesize), filename,
    ])]
    changes['Checksums-Sha256'] = [u' '.join([
        sha256, str(filesize), filename,
    ])]

    return changes


if '__main__' == __name__:
    try:
        main()
    except Exception:
        logger.exception("Unhandled error:")
        import pdb
        pdb.post_mortem()
