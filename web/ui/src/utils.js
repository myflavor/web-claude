export function fmtTime(iso) {
  try {
    const d = new Date(iso);
    if (Number.isNaN(d.getTime())) return "";
    return d.toLocaleString();
  } catch {
    return "";
  }
}

export function escapeHtml(s) {
  return String(s)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

export function initialOf(title) {
  const t = (title || "C").trim();
  return (t[0] || "C").toUpperCase();
}

export function displayPath(projectsRoot, rel) {
  const root = (projectsRoot || "").replace(/\/+$/, "") || "";
  const r = (rel || "").replace(/^\/+/, "");
  if (!root) return r ? `/${r}` : "/";
  return r ? `${root}/${r}` : root;
}

export function folderTitle(projectsRoot, rel, absFallback) {
  const r = (rel || "").replace(/^\/+|\/+$/g, "");
  if (r) {
    const parts = r.split("/");
    return parts[parts.length - 1] || "projects";
  }
  if (absFallback) {
    const parts = String(absFallback).replace(/\/+$/, "").split("/");
    return parts[parts.length - 1] || "projects";
  }
  if (projectsRoot) {
    const parts = projectsRoot.replace(/\/+$/, "").split("/");
    return parts[parts.length - 1] || "projects";
  }
  return "projects";
}

export function parseArgLine(line) {
  const out = [];
  const s = String(line || "").trim();
  if (!s) return out;
  let cur = "";
  let quote = null;
  for (let i = 0; i < s.length; i++) {
    const ch = s[i];
    if (quote) {
      if (ch === quote) quote = null;
      else cur += ch;
      continue;
    }
    if (ch === '"' || ch === "'") {
      quote = ch;
      continue;
    }
    if (/\s/.test(ch)) {
      if (cur) {
        out.push(cur);
        cur = "";
      }
      continue;
    }
    cur += ch;
  }
  if (cur) out.push(cur);
  return out;
}
