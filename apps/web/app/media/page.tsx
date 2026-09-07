import { ClipActions, ClipForm, MediaAssetForm, TranscriptForm } from "@/components/media-actions";
import { getMediaData, msLabel } from "@/lib/media-api";

export default async function MediaPage() {
  const data = await getMediaData();
  const rendered = data.clips.filter((clip) => clip.status === "rendered").length;
  const approved = data.clips.filter((clip) => clip.status === "approved" || clip.status === "queued" || clip.status === "rendering").length;
  const transcribed = data.assets.filter((asset) => asset.transcriptText || asset.transcriptSegments.length > 0).length;
  const clipsByAsset = new Map<string, typeof data.clips>();
  for (const clip of data.clips) clipsByAsset.set(clip.mediaAssetId, [...(clipsByAsset.get(clip.mediaAssetId) ?? []), clip]);

  return <div className="page-wrap media-page">
    <header className="topbar compact-topbar"><div><span className="eyebrow">Media Studio</span><h1>Turn recordings into approved, renderable developer content.</h1><p>Track source recordings, transcripts and clip candidates, then render approved clips with the worker-side FFmpeg pipeline.</p></div><a className="button ghost" href="/content">Content Studio →</a></header>
    {!data.connected && <div className="notice"><strong>Media API is not connected.</strong><span>Apply migrations and start the Go API/worker to use rendering.</span></div>}
    <section className="metrics-grid">
      <article className="metric-card"><span className="eyebrow">Assets</span><strong className="metric-value">{data.assets.length}</strong><span className="muted">recordings tracked</span></article>
      <article className="metric-card"><span className="eyebrow">Transcribed</span><strong className="metric-value">{transcribed}</strong><span className="muted">ready for clip review</span></article>
      <article className="metric-card"><span className="eyebrow">Approved / queued</span><strong className="metric-value">{approved}</strong><span className="muted">human-approved clips</span></article>
      <article className="metric-card"><span className="eyebrow">Rendered</span><strong className="metric-value">{rendered}</strong><span className="muted">FFmpeg outputs</span></article>
    </section>
    <section className="panel"><div className="panel-head"><div><span className="eyebrow">Source Media</span><h2>Add a recording</h2></div><span className="muted panel-note">Local render paths are resolved inside DEVRELOS_MEDIA_ROOT.</span></div><div className="media-form-wrap"><MediaAssetForm /></div></section>
    <section className="media-assets-grid">
      {data.assets.length === 0 ? <article className="panel media-empty"><span className="eyebrow">Media Pipeline</span><h2>No recordings yet</h2><p>Add a local recording path such as <code>recordings/demo.mp4</code>, then attach a transcript and create clip candidates.</p></article> : data.assets.map((asset) => {
        const clips = clipsByAsset.get(asset.id) ?? [];
        return <article className="panel media-asset-card" key={asset.id}>
          <div className="panel-head"><div><span className="eyebrow">{asset.mediaType} · {asset.status}</span><h2>{asset.title}</h2></div><span className="pill neutral">{asset.sourcePath || "remote reference"}</span></div>
          {asset.sourceUrl && <a className="text-button" href={asset.sourceUrl} target="_blank">Open source ↗</a>}
          <div className="media-transcript"><div className="panel-head"><div><span className="eyebrow">Transcript</span><h3>{asset.transcriptText ? "Imported" : "Not imported"}</h3></div></div>{asset.transcriptText && <p>{asset.transcriptText.length > 480 ? `${asset.transcriptText.slice(0, 480)}…` : asset.transcriptText}</p>}<TranscriptForm asset={asset} /></div>
          <div className="media-clip-create"><span className="eyebrow">New Clip Candidate</span><ClipForm asset={asset} /></div>
          <div className="media-clips-list">{clips.length === 0 ? <p className="empty-copy">No clip candidates for this recording.</p> : clips.map((clip) => <div className="media-clip-row" key={clip.id}><div><div className="media-clip-head"><strong>{clip.title}</strong><span className="score">{clip.score}</span></div><span className="muted">{msLabel(clip.startMs)}–{msLabel(clip.endMs)} · {clip.aspectRatio} · {clip.status}</span>{clip.rationale && <p>{clip.rationale}</p>}{clip.outputPath && <code>{clip.outputPath}</code>}</div><ClipActions clip={clip} /></div>)}</div>
        </article>;
      })}
    </section>
  </div>;
}
