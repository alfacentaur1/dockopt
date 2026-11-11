// Web demo for dockopt: the Go core compiled to WebAssembly does the work.

const EXAMPLES = {
  go: {
    files: ["go.mod", "go.sum", "main.go"],
    src: `FROM golang:1.22
MAINTAINER Filip Kopecky
WORKDIR /src

RUN apt-get update
RUN apt-get install -y curl

COPY . .
RUN go build -o /usr/local/bin/api .

EXPOSE 8080
CMD /usr/local/bin/api
`,
  },
  node: {
    files: ["package.json", "package-lock.json", "index.js"],
    src: `FROM node:latest
WORKDIR /app
COPY . .
RUN npm ci
RUN npm run build
ADD config.json /app/config.json
CMD npm start
`,
  },
  python: {
    files: ["requirements.txt", "app.py"],
    src: `FROM python:3.12-slim
RUN apt-get update && apt-get -y upgrade
RUN apt-get install -y build-essential
COPY . /srv
RUN cd /srv && pip install -r requirements.txt
CMD ["python", "/srv/app.py"]
`,
  },
  clean: {
    files: ["go.mod", "go.sum", "main.go"],
    src: `# syntax=docker/dockerfile:1
FROM golang:1.22 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/api .

FROM gcr.io/distroless/static:nonroot
COPY --from=build /out/api /api
ENTRYPOINT ["/api"]
`,
  },
};

const $ = (id) => document.getElementById(id);
const editor = $("editor");
const example = $("example");
const hasIgnore = $("has-ignore");
let files = EXAMPLES.go.files;
let lastFix = null;

function options() {
  return { dockerignore: hasIgnore.checked ? ".git\nnode_modules\n" : null, files, ignore: [] };
}

function escape(s) {
  return s.replace(/[&<>]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;" })[c]);
}

function renderFindings(findings) {
  const box = $("findings");
  $("count").textContent = findings.length;
  if (findings.length === 0) {
    box.innerHTML = '<p class="ok">No issues. Nice Dockerfile.</p>';
    return;
  }
  box.innerHTML = "";
  for (const f of findings) {
    const row = document.createElement("button");
    row.className = "finding";
    row.innerHTML =
      `<span class="line">${f.line}</span>` +
      `<span class="sev ${f.severity}">${f.severity}</span>` +
      `<span class="rule">${f.rule}</span>` +
      `<span>${escape(f.message)}${f.fixable ? '<span class="fixable">fixable</span>' : ""}</span>`;
    row.title = "Jump to line " + f.line;
    row.addEventListener("click", () => selectLine(f.line));
    box.appendChild(row);
  }
}

function renderDiff(diff) {
  if (!diff) return '<span class="muted">Nothing to change.</span>';
  return diff
    .split("\n")
    .map((l) => {
      const e = escape(l);
      if (l.startsWith("@@")) return `<span class="hunk">${e}</span>`;
      if (l.startsWith("+") && !l.startsWith("+++")) return `<span class="add">${e}</span>`;
      if (l.startsWith("-") && !l.startsWith("---")) return `<span class="del">${e}</span>`;
      return e;
    })
    .join("\n");
}

function selectLine(n) {
  const lines = editor.value.split("\n");
  let start = 0;
  for (let i = 0; i < n - 1 && i < lines.length; i++) start += lines[i].length + 1;
  editor.focus();
  editor.setSelectionRange(start, start + (lines[n - 1] || "").length);
}

function run() {
  const a = JSON.parse(dockopt.analyze(editor.value, options()));
  if (a.error) {
    $("findings").innerHTML = `<p class="err">${escape(a.error)}</p>`;
    $("count").textContent = "!";
    $("apply").disabled = true;
    return;
  }
  renderFindings(a.findings || []);

  lastFix = JSON.parse(dockopt.fix(editor.value, options()));
  $("fixed").textContent = lastFix.output || editor.value;
  $("diff").innerHTML = renderDiff(lastFix.diff);
  $("apply").disabled = !(lastFix.applied && lastFix.applied.length);
}

let timer;
function schedule() {
  clearTimeout(timer);
  timer = setTimeout(run, 150);
}

function loadExample(name) {
  files = EXAMPLES[name].files;
  editor.value = EXAMPLES[name].src;
  hasIgnore.checked = name === "clean";
  run();
}

document.querySelectorAll('[role="tab"]').forEach((tab) =>
  tab.addEventListener("click", () => {
    document.querySelectorAll('[role="tab"]').forEach((t) => t.setAttribute("aria-selected", String(t === tab)));
    for (const id of ["findings", "fixed", "diff"]) $(id).hidden = id !== tab.dataset.tab;
  }),
);

$("apply").addEventListener("click", () => {
  if (lastFix && lastFix.output) {
    editor.value = lastFix.output;
    run();
  }
});
editor.addEventListener("input", schedule);
hasIgnore.addEventListener("change", run);
example.addEventListener("change", () => loadExample(example.value));

window.addEventListener("dockopt-ready", () => {
  const rules = JSON.parse(dockopt.rules());
  $("rules-count").textContent = `${rules.length} rules, ${rules.filter((r) => r.fixable).length} with automatic fixes`;
  loadExample("go");
});

const go = new Go();
WebAssembly.instantiateStreaming(fetch("dockopt.wasm"), go.importObject)
  .then((r) => go.run(r.instance))
  .catch((err) => {
    $("findings").innerHTML = `<p class="err">Couldn't load the analyzer: ${escape(String(err))}</p>`;
  });
