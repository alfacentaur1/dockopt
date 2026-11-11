package rules

import "testing"

func TestDO009Check(t *testing.T) {
	assertFindings(t, "DO009", ctxFor(t, "FROM python:3.12\nRUN pip install -r requirements.txt\n"), 1)
	assertFindings(t, "DO009", ctxFor(t, "FROM python:3.12\nRUN python -m pip install flask\n"), 1)
	assertFindings(t, "DO009", ctxFor(t, "FROM python:3.12\nRUN pip install --no-cache-dir flask\n"), 0)
	assertFindings(t, "DO009", ctxFor(t, "FROM python:3.12\nENV PIP_NO_CACHE_DIR=1\nRUN pip install flask\n"), 0)
	assertFindings(t, "DO009", ctxFor(t, "FROM python:3.12\nRUN --mount=type=cache,target=/root/.cache/pip pip install flask\n"), 0)
}

func TestDO009Fix(t *testing.T) {
	assertFixed(t, "DO009",
		"FROM python:3.12\nRUN pip3 install -U pip && pip3 install -r requirements.txt\n",
		"FROM python:3.12\nRUN pip3 install --no-cache-dir -U pip && pip3 install --no-cache-dir -r requirements.txt\n")
}
