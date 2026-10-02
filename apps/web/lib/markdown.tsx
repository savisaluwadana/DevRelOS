import type { ReactNode } from "react";

// A deliberately small Markdown renderer for the operator documentation.
//
// It covers exactly the constructs those documents use - headings, tables,
// fenced code, bullet and numbered lists, blockquotes, rules, and inline bold,
// code and links - rather than pulling in a Markdown library and its
// dependency tree for six files we control. Output is React elements, never
// raw HTML, so document text can never inject markup.

export type Heading = { depth: number; text: string; id: string };

export function slugify(text: string): string {
  return text
    .toLowerCase()
    .replace(/`/g, "")
    .replace(/[^a-z0-9\s-]/g, "")
    .trim()
    .replace(/\s+/g, "-")
    .slice(0, 80);
}

/** Inline: `code`, **bold**, [text](href). Code is matched first so markup inside it stays literal. */
function renderInline(text: string, keyPrefix: string): ReactNode[] {
  const out: ReactNode[] = [];
  const pattern = /(`[^`]+`)|(\*\*[^*]+\*\*)|(\[[^\]]+\]\([^)\s]+\))/g;
  let last = 0;
  let match: RegExpExecArray | null;
  let index = 0;

  while ((match = pattern.exec(text)) !== null) {
    if (match.index > last) out.push(text.slice(last, match.index));
    const token = match[0];
    const key = `${keyPrefix}-i${index++}`;

    if (token.startsWith("`")) {
      out.push(<code key={key}>{token.slice(1, -1)}</code>);
    } else if (token.startsWith("**")) {
      out.push(<strong key={key}>{token.slice(2, -2)}</strong>);
    } else {
      const split = token.indexOf("](");
      const label = token.slice(1, split);
      const href = token.slice(split + 2, -1);
      // Cross-document links point at other bundled guides; rewrite them to
      // the in-app route so the guide is self-contained.
      const internal = /^([A-Z_]+)\.md(#.*)?$/.exec(href);
      if (internal) {
        const slug = docSlugForFile(internal[1]);
        out.push(
          <a key={key} href={slug ? `/guide?doc=${slug}${internal[2] ?? ""}` : "#"}>
            {label}
          </a>
        );
      } else if (/^https?:\/\//.test(href) || href.startsWith("#") || href.startsWith("/")) {
        out.push(
          <a key={key} href={href} {...(href.startsWith("http") ? { target: "_blank", rel: "noreferrer" } : {})}>
            {label}
          </a>
        );
      } else {
        // A relative repository path has no meaning in the browser.
        out.push(<span key={key}>{label}</span>);
      }
    }
    last = match.index + token.length;
  }
  if (last < text.length) out.push(text.slice(last));
  return out;
}

const fileToSlug: Record<string, string> = {
  WORKFLOW: "workflow",
  OPERATOR_GUIDE: "operator",
  INTEGRATIONS: "integrations",
  RUNNING: "running",
  SECURITY: "security",
  MCP: "mcp",
};

function docSlugForFile(name: string): string | undefined {
  return fileToSlug[name];
}

function splitRow(line: string): string[] {
  return line
    .replace(/^\|/, "")
    .replace(/\|$/, "")
    .split("|")
    .map((cell) => cell.trim());
}

function isSeparatorRow(line: string): boolean {
  return /^\|?[\s:-]+\|[\s:|-]*$/.test(line) && line.includes("-");
}

export function renderMarkdown(markdown: string): { body: ReactNode[]; headings: Heading[] } {
  const lines = markdown.replace(/\r\n/g, "\n").split("\n");
  const body: ReactNode[] = [];
  const headings: Heading[] = [];
  let key = 0;
  const next = () => `b${key++}`;

  for (let i = 0; i < lines.length; i += 1) {
    const line = lines[i];

    // Fenced code
    if (line.startsWith("```")) {
      const language = line.slice(3).trim();
      const buffer: string[] = [];
      i += 1;
      while (i < lines.length && !lines[i].startsWith("```")) {
        buffer.push(lines[i]);
        i += 1;
      }
      const content = buffer.join("\n");
      // Mermaid needs a diagram runtime the console does not ship, so its
      // source would otherwise sit in the page as ~30 lines of unreadable
      // syntax. Collapse it instead of pretending to render it.
      if (language === "mermaid") {
        body.push(
          <details key={next()} className="guide-diagram">
            <summary>Diagram source (Mermaid — renders on GitHub)</summary>
            <pre className="guide-code guide-code-mermaid">
              <code>{content}</code>
            </pre>
          </details>
        );
        continue;
      }
      body.push(
        <pre key={next()} className={`guide-code${language ? ` guide-code-${language}` : ""}`}>
          {language ? <span className="guide-code-lang">{language}</span> : null}
          <code>{content}</code>
        </pre>
      );
      continue;
    }

    // Horizontal rule
    if (/^---+$/.test(line.trim())) {
      body.push(<hr key={next()} className="guide-rule" />);
      continue;
    }

    // Heading
    const heading = /^(#{1,6})\s+(.*)$/.exec(line);
    if (heading) {
      const depth = heading[1].length;
      const text = heading[2].trim();
      const id = slugify(text);
      if (depth <= 3) headings.push({ depth, text, id });
      const Tag = `h${Math.min(depth + 1, 6)}` as "h2";
      body.push(
        <Tag key={next()} id={id} className={`guide-h guide-h${depth}`}>
          {renderInline(text, id)}
        </Tag>
      );
      continue;
    }

    // Table
    if (line.trim().startsWith("|") && i + 1 < lines.length && isSeparatorRow(lines[i + 1])) {
      const header = splitRow(line);
      i += 2;
      const rows: string[][] = [];
      while (i < lines.length && lines[i].trim().startsWith("|")) {
        rows.push(splitRow(lines[i]));
        i += 1;
      }
      i -= 1;
      const tableKey = next();
      body.push(
        <div key={tableKey} className="guide-table-wrap">
          <table className="guide-table">
            <thead>
              <tr>
                {header.map((cell, c) => (
                  <th key={`${tableKey}-h${c}`}>{renderInline(cell, `${tableKey}-h${c}`)}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {rows.map((row, r) => (
                <tr key={`${tableKey}-r${r}`}>
                  {row.map((cell, c) => (
                    <td key={`${tableKey}-r${r}c${c}`}>{renderInline(cell, `${tableKey}-r${r}c${c}`)}</td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      );
      continue;
    }

    // Blockquote
    if (line.startsWith("> ")) {
      const buffer: string[] = [];
      while (i < lines.length && (lines[i].startsWith("> ") || lines[i].trim() === ">")) {
        buffer.push(lines[i].replace(/^>\s?/, ""));
        i += 1;
      }
      i -= 1;
      const quoteKey = next();
      body.push(
        <blockquote key={quoteKey} className="guide-quote">
          {renderInline(buffer.join(" ").trim(), quoteKey)}
        </blockquote>
      );
      continue;
    }

    // Lists
    const bullet = /^\s*[-*]\s+(.*)$/.exec(line);
    const numbered = /^\s*\d+\.\s+(.*)$/.exec(line);
    if (bullet || numbered) {
      const ordered = Boolean(numbered);
      const items: string[] = [];
      while (i < lines.length) {
        const current = lines[i];
        const nextBullet = /^\s*[-*]\s+(.*)$/.exec(current);
        const nextNumber = /^\s*\d+\.\s+(.*)$/.exec(current);
        if (nextBullet && !ordered) items.push(nextBullet[1]);
        else if (nextNumber && ordered) items.push(nextNumber[1]);
        else if (/^\s{2,}\S/.test(current) && items.length > 0) {
          // A wrapped continuation line belongs to the previous item.
          items[items.length - 1] += ` ${current.trim()}`;
        } else break;
        i += 1;
      }
      i -= 1;
      const listKey = next();
      const children = items.map((item, index) => (
        <li key={`${listKey}-${index}`}>{renderInline(item, `${listKey}-${index}`)}</li>
      ));
      body.push(
        ordered ? (
          <ol key={listKey} className="guide-list">{children}</ol>
        ) : (
          <ul key={listKey} className="guide-list">{children}</ul>
        )
      );
      continue;
    }

    if (line.trim() === "") continue;

    // Paragraph: absorb following non-blank, non-structural lines.
    const buffer = [line.trim()];
    while (i + 1 < lines.length) {
      const peek = lines[i + 1];
      if (
        peek.trim() === "" ||
        peek.startsWith("```") ||
        peek.startsWith("#") ||
        peek.startsWith("> ") ||
        peek.trim().startsWith("|") ||
        /^---+$/.test(peek.trim()) ||
        /^\s*[-*]\s+/.test(peek) ||
        /^\s*\d+\.\s+/.test(peek)
      ) {
        break;
      }
      buffer.push(peek.trim());
      i += 1;
    }
    const paraKey = next();
    body.push(
      <p key={paraKey} className="guide-p">
        {renderInline(buffer.join(" "), paraKey)}
      </p>
    );
  }

  return { body, headings };
}
