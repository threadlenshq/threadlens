// Renders the AI-generated "Overall Assessment" string to safe HTML.
//
// The model is not given a fixed output format, so assessments arrive in a few
// shapes: plain prose, double-newline paragraphs, newline-separated "1." lists,
// and - most commonly - a single unbroken block with inline "(N) **Heading** -
// description" enumeration. This renderer normalises all of those into a lead
// paragraph + ordered list + trailing summary paragraph so a dense block of
// "(1)...(2)...(3)..." doesn't render as one wall of text.

const escapeHtml = (s) =>
  s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');

// Escape, then promote **bold** to <strong> (AI content can't inject markup).
const inline = (s) => escapeHtml(s).replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>');

const orderedList = (items) =>
  '<ol>' + items.map((s) => `<li>${inline(s.trim())}</li>`).join('') + '</ol>';

// Sentence cues that mark where closing/summary prose begins after an inline
// list. Used to peel a trailing paragraph off the final enumerated item so the
// summary isn't swallowed into it.
const TAIL_CUE =
  /\.\s+(?=(?:The (?:strongest|biggest|most|top)|Overall|Together|Collectively|Combined|Across|Notably|In (?:summary|total)|Each |All |These |Both )\b)/;

function renderBlock(block) {
  const lines = block.split('\n');

  // Newline-separated numbered list: "1. ...\n2. ...\n3. ..."
  if (lines.length > 1 && lines.every((l) => /^\d+\.\s/.test(l.trim()))) {
    return orderedList(lines.map((l) => l.replace(/^\d+\.\s*/, '')));
  }

  // Inline enumeration: "lead: (1) ... (2) ... (3) ... tail". Only treat it as a
  // list when the markers run sequentially from 1, so a stray "(2)" reference
  // isn't mistaken for a list.
  const markers = [...block.matchAll(/\((\d+)\)\s*/g)];
  const sequential =
    markers.length >= 2 && markers.every((m, i) => Number(m[1]) === i + 1);

  if (sequential) {
    const lead = block.slice(0, markers[0].index).trim();
    const items = markers.map((m, i) => {
      const start = m.index + m[0].length;
      const end = i + 1 < markers.length ? markers[i + 1].index : block.length;
      return block.slice(start, end).trim();
    });

    // Peel trailing summary prose off the last item, if present.
    let tail = '';
    const last = items[items.length - 1];
    const cut = last.match(TAIL_CUE);
    if (cut) {
      items[items.length - 1] = last.slice(0, cut.index + 1).trim();
      tail = last.slice(cut.index + cut[0].length).trim();
    }

    const out = [];
    if (lead) out.push(`<p>${inline(lead)}</p>`);
    out.push(orderedList(items));
    if (tail) out.push(`<p>${inline(tail)}</p>`);
    return out.join('');
  }

  return `<p>${inline(block).replace(/\n/g, '<br>')}</p>`;
}

export function renderAssessment(text) {
  if (!text) return '';
  return text
    .split(/\n{2,}/)
    .map((b) => b.trim())
    .filter(Boolean)
    .map(renderBlock)
    .join('');
}
