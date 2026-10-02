import Link from "next/link";
import { guideDocs, findGuideDoc } from "@/lib/generated-docs";
import { renderMarkdown } from "@/lib/markdown";

export const metadata = {
  title: "Guide — DevRelOS"
};

type Props = {
  searchParams: Promise<{ doc?: string }>;
};

export default async function GuidePage({ searchParams }: Props) {
  const params = await searchParams;
  const doc = findGuideDoc(params.doc);
  const { body, headings } = renderMarkdown(doc.markdown);
  const sections = headings.filter((heading) => heading.depth === 2);

  return (
    <div className="page-wrap guide-page">
      <header className="topbar compact-topbar">
        <div>
          <span className="eyebrow">Guide</span>
          <h1>How to run DevRelOS.</h1>
          <p>
            The operator documentation, in the console. Start with the Workflow Guide for how the loops
            connect, then use the Operator Guide as a per-workspace reference.
          </p>
        </div>
        <Link className="button ghost" href="/">← Command Center</Link>
      </header>

      <nav className="guide-picker" aria-label="Documentation">
        {guideDocs.map((entry) => (
          <Link
            key={entry.slug}
            href={`/guide?doc=${entry.slug}`}
            className={`guide-pick${entry.slug === doc.slug ? " is-active" : ""}`}
            aria-current={entry.slug === doc.slug ? "page" : undefined}
          >
            <strong>{entry.title}</strong>
            <span className="muted">{entry.blurb}</span>
          </Link>
        ))}
      </nav>

      <section className="guide-layout">
        {sections.length > 0 && (
          <aside className="panel guide-toc" aria-label={`Sections of ${doc.title}`}>
            <div className="panel-head">
              <div>
                <span className="eyebrow">On this page</span>
                <h2>{doc.title}</h2>
              </div>
            </div>
            <ol className="guide-toc-list">
              {sections.map((heading) => (
                <li key={heading.id}>
                  <a href={`#${heading.id}`}>{heading.text}</a>
                </li>
              ))}
            </ol>
          </aside>
        )}

        <article className="panel guide-body">
          {body}
        </article>
      </section>
    </div>
  );
}
