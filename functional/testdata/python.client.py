#!/usr/bin/env python3
"""Official fluent-logger (Python) client, driven by the functional tests.

fluent-logger always uses Message mode and packs [tag, timestamp, record].
The FLUENT_NANOSECOND_PRECISION switch selects the timestamp format:
  - "0" (default): integer epoch seconds, the historical format
  - "1": EventTime ext, the contemporary format
"""

import os
import sys

from fluent.sender import FluentSender

host = os.environ.get("FLUENT_HOST", "host.docker.internal")
port = int(os.environ.get("FLUENT_PORT", "24224"))
tag = os.environ.get("FLUENT_TAG", "testpython")
nanosecond_precision = os.environ.get("FLUENT_NANOSECOND_PRECISION", "0") == "1"

sender = FluentSender(tag, host=host, port=port, nanosecond_precision=nanosecond_precision)
ok = sender.emit("event", {"client": "python", "value": 42})
sender.close()

if not ok:
    print("emit failed: %s" % (sender.last_error,), file=sys.stderr)
    sys.exit(1)
