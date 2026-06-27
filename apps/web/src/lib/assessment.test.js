import { describe, expect, it } from 'vitest';
import { renderAssessment } from './assessment.js';

// The real assessment string for research report #13: one unbroken block with
// inline "(N) **Heading** - description" enumeration, no newlines.
const REPORT_13 =
  'This research reveals a foundational market gap: builders have direct access to customer research gold on Reddit, Twitter, and niche forums, but extracting and acting on that signal is manually intensive, time-consuming, and error-prone. The unified opportunity space can be summarized as a sequence of pains that unfold from idea to scale: ' +
  '(1) **Pre-Build Validation** - Founders spend 3-6 months uncertain whether an idea is real, manually searching Reddit with no synthesis mechanism. ' +
  '(2) **Post-Launch User Acquisition** - Builders ship working products but struggle to find their first 10-100 users because they don\'t know where their customers congregate. ' +
  '(3) **Product Intelligence** - Once users exist, founders can\'t systematically extract what customers actually want from scattered feedback. ' +
  '(4) **Growth Playbook** - Builders stuck at 100-1000 users lack peer networks and growth strategies. ' +
  '(5) **Infrastructure & Operations** - Self-hosting and AI agent management remain brittle and costly, forcing technical builders into SaaS dependency. ' +
  'The strongest near-term opportunities are: **ThreadLens** (Reddit research automation), **First User Matchmaking** (beta-tester platform), and **Self-Hosted Deployment Automation** (Docker template library). ' +
  'Each has 50-150+ validation posts with clear workarounds builders have already constructed, indicating both pain intensity and willingness to pay.';

describe('renderAssessment', () => {
  it('returns empty string for empty input', () => {
    expect(renderAssessment('')).toBe('');
    expect(renderAssessment(null)).toBe('');
    expect(renderAssessment(undefined)).toBe('');
  });

  it('escapes HTML so AI content cannot inject markup', () => {
    const html = renderAssessment('A <script>alert(1)</script> & "quote".');
    expect(html).not.toContain('<script>');
    expect(html).toContain('&lt;script&gt;');
    expect(html).toContain('&amp;');
  });

  it('converts **bold** to <strong>', () => {
    expect(renderAssessment('This is **important** text.')).toContain(
      '<strong>important</strong>'
    );
  });

  it('wraps a plain paragraph in <p>', () => {
    expect(renderAssessment('Just one sentence.')).toBe('<p>Just one sentence.</p>');
  });

  it('keeps double-newline-separated paragraphs distinct', () => {
    const html = renderAssessment('First para.\n\nSecond para.');
    expect(html).toBe('<p>First para.</p><p>Second para.</p>');
  });

  it('renders a newline-separated numbered list as <ol>', () => {
    const html = renderAssessment('1. First item\n2. Second item\n3. Third item');
    expect(html).toBe(
      '<ol><li>First item</li><li>Second item</li><li>Third item</li></ol>'
    );
  });

  it('breaks an inline "(N) ..." enumeration into an ordered list', () => {
    const html = renderAssessment(
      'Three pains: (1) one thing. (2) another thing. (3) a third thing.'
    );
    expect(html).toContain('<ol>');
    expect((html.match(/<li>/g) || []).length).toBe(3);
    // The "(N)" markers themselves are stripped (the <ol> supplies numbering)
    expect(html).not.toContain('(1)');
    expect(html).not.toContain('(2)');
  });

  it('splits report #13 into a lead paragraph, a 5-item list, and a closing paragraph', () => {
    const html = renderAssessment(REPORT_13);
    const liCount = (html.match(/<li>/g) || []).length;
    const pCount = (html.match(/<p>/g) || []).length;

    expect(liCount).toBe(5);
    // Lead-in prose before "(1)" and the closing summary after the list.
    expect(pCount).toBe(2);

    // Lead paragraph holds the intro, not the first list item.
    expect(html).toMatch(/^<p>This research reveals/);
    // Headings survive as <strong> inside list items.
    expect(html).toContain('<strong>Pre-Build Validation</strong>');
    // The "(5)" item must NOT swallow the closing summary prose.
    expect(html).toContain('<strong>Infrastructure &amp; Operations</strong>');
    expect(html).toContain('<p>The strongest near-term opportunities are:');
    // Closing analysis sentence stays in the trailing paragraph.
    expect(html).toContain('Each has 50-150+ validation posts');
  });

  it('does not treat a stray "(2)" reference as an enumeration', () => {
    const html = renderAssessment('See figure (2) for details. No real list here.');
    expect(html).toBe('<p>See figure (2) for details. No real list here.</p>');
  });
});
