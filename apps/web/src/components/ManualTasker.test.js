// @vitest-environment jsdom
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';
import ManualTasker from './ManualTasker.svelte';
import { manualScout, manualScoutCommit } from '../lib/api.js';

vi.mock('../lib/api.js', () => ({
  manualScout: vi.fn(),
  manualScoutCommit: vi.fn(),
}));

vi.mock('../lib/format.js', () => ({
  scoreColor: vi.fn(() => '#98c379'),
}));

const TEST_PROJECT_ID = 'project-123';

describe('ManualTasker', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  it('renders input form with URL input, platform selector, and submit button', () => {
    render(ManualTasker, { props: { projectId: TEST_PROJECT_ID } });

    expect(screen.getByLabelText('URL')).toBeTruthy();
    expect(screen.getByRole('button', { name: /reddit/i })).toBeTruthy();
    expect(screen.getByRole('button', { name: /bluesky/i })).toBeTruthy();
    expect(screen.getByRole('button', { name: /scout url/i })).toBeTruthy();
  });

  it('disables submit button when URL is empty', () => {
    render(ManualTasker, { props: { projectId: TEST_PROJECT_ID } });

    const submitBtn = screen.getByRole('button', { name: /scout url/i });
    expect(submitBtn.disabled).toBe(true);
  });

  it('calls manualScout on submit with valid URL and platform', async () => {
    manualScout.mockResolvedValue({ status: 'saved', post: { title: 'Test Post', url: 'https://reddit.com/r/test' } });

    render(ManualTasker, { props: { projectId: TEST_PROJECT_ID } });

    const urlInput = screen.getByLabelText('URL');
    await fireEvent.input(urlInput, { target: { value: 'https://reddit.com/r/test' } });

    const submitBtn = screen.getByRole('button', { name: /scout url/i });
    await fireEvent.click(submitBtn);

    expect(manualScout).toHaveBeenCalledTimes(1);
    expect(manualScout).toHaveBeenCalledWith(TEST_PROJECT_ID, 'https://reddit.com/r/test', 'reddit');
  });

  it('shows loading state during submit', async () => {
    let resolve;
    const promise = new Promise((res) => { resolve = res; });
    manualScout.mockReturnValue(promise);

    render(ManualTasker, { props: { projectId: TEST_PROJECT_ID } });

    const urlInput = screen.getByLabelText('URL');
    await fireEvent.input(urlInput, { target: { value: 'https://reddit.com/r/test' } });

    const submitBtn = screen.getByRole('button', { name: /scout url/i });
    await fireEvent.click(submitBtn);

    expect(screen.getByText(/scouting.../i)).toBeTruthy();
    expect(screen.queryByText(/scout url/i)).toBeNull();

    resolve({ status: 'saved', post: { title: 'Test' } });
    await waitFor(() => expect(screen.queryByText(/scouting.../i)).toBeNull());
  });

  it('shows "saved" success state with green display', async () => {
    manualScout.mockResolvedValue({
      status: 'saved',
      post: { title: 'Great Post', url: 'https://reddit.com/r/test', final_score: 7.5 },
      score: 7.5,
    });

    render(ManualTasker, { props: { projectId: TEST_PROJECT_ID } });

    const urlInput = screen.getByLabelText('URL');
    await fireEvent.input(urlInput, { target: { value: 'https://reddit.com/r/test' } });

    const submitBtn = screen.getByRole('button', { name: /scout url/i });
    await fireEvent.click(submitBtn);

    await waitFor(() => expect(screen.getByText('Saved')).toBeTruthy());
    expect(screen.getByText('Great Post')).toBeTruthy();
  });

  it('shows "already_scouted" state with blue info display', async () => {
    manualScout.mockResolvedValue({
      status: 'already_scouted',
      post: { title: 'Old Post', url: 'https://reddit.com/r/test', final_score: 5.0 },
    });

    render(ManualTasker, { props: { projectId: TEST_PROJECT_ID } });

    const urlInput = screen.getByLabelText('URL');
    await fireEvent.input(urlInput, { target: { value: 'https://reddit.com/r/test' } });

    const submitBtn = screen.getByRole('button', { name: /scout url/i });
    await fireEvent.click(submitBtn);

    await waitFor(() => expect(screen.getByText('Already Scouted')).toBeTruthy());
    expect(screen.getByText('Old Post')).toBeTruthy();
  });

  it('shows "needs_decision" with Keep/Exclude buttons and post data', async () => {
    manualScout.mockResolvedValue({
      status: 'needs_decision',
      post: { title: 'Borderline Post', url: 'https://reddit.com/r/test', post_score: 1.5 },
      score: 1.5,
    });

    render(ManualTasker, { props: { projectId: TEST_PROJECT_ID } });

    const urlInput = screen.getByLabelText('URL');
    await fireEvent.input(urlInput, { target: { value: 'https://reddit.com/r/test' } });

    const submitBtn = screen.getByRole('button', { name: /scout url/i });
    await fireEvent.click(submitBtn);

    await waitFor(() => expect(screen.getByText('Needs Decision')).toBeTruthy());
    expect(screen.getByText('Borderline Post')).toBeTruthy();
    expect(screen.getByRole('button', { name: /keep/i })).toBeTruthy();
    expect(screen.getByRole('button', { name: /exclude/i })).toBeTruthy();
  });

  it('shows "filtered" state with filter reason', async () => {
    manualScout.mockResolvedValue({
      status: 'filtered',
      post: { title: 'Spam Post', url: 'https://reddit.com/r/test' },
      filter: { reason: 'Spam', explanation: 'Detected as promotional content' },
      reason: 'Spam',
      explanation: 'Detected as promotional content',
    });

    render(ManualTasker, { props: { projectId: TEST_PROJECT_ID } });

    const urlInput = screen.getByLabelText('URL');
    await fireEvent.input(urlInput, { target: { value: 'https://reddit.com/r/test' } });

    const submitBtn = screen.getByRole('button', { name: /scout url/i });
    await fireEvent.click(submitBtn);

    await waitFor(() => expect(screen.getByText('Filtered')).toBeTruthy());
    expect(screen.getByText('Spam')).toBeTruthy();
    expect(screen.getByText('Detected as promotional content')).toBeTruthy();
  });

  it('shows "error" state when API returns error status', async () => {
    manualScout.mockResolvedValue({
      status: 'error',
      error: 'Something went wrong',
      message: 'Something went wrong',
    });

    render(ManualTasker, { props: { projectId: TEST_PROJECT_ID } });

    const urlInput = screen.getByLabelText('URL');
    await fireEvent.input(urlInput, { target: { value: 'https://reddit.com/r/test' } });

    const submitBtn = screen.getByRole('button', { name: /scout url/i });
    await fireEvent.click(submitBtn);

    await waitFor(() => expect(screen.getByText('Error')).toBeTruthy());
    expect(screen.getByText('Something went wrong')).toBeTruthy();
  });

  it('calls manualScoutCommit when Keep button is clicked', async () => {
    const postData = { title: 'Borderline Post', url: 'https://reddit.com/r/test', post_score: 1.5 };
    manualScout.mockResolvedValue({
      status: 'needs_decision',
      post: postData,
      score: 1.5,
    });
    manualScoutCommit.mockResolvedValue({ status: 'saved', post: postData });

    render(ManualTasker, { props: { projectId: TEST_PROJECT_ID } });

    const urlInput = screen.getByLabelText('URL');
    await fireEvent.input(urlInput, { target: { value: 'https://reddit.com/r/test' } });

    const submitBtn = screen.getByRole('button', { name: /scout url/i });
    await fireEvent.click(submitBtn);

    await waitFor(() => expect(screen.getByRole('button', { name: /keep/i })).toBeTruthy());

    const keepBtn = screen.getByRole('button', { name: /keep/i });
    await fireEvent.click(keepBtn);

    expect(manualScoutCommit).toHaveBeenCalledTimes(1);
    expect(manualScoutCommit).toHaveBeenCalledWith(TEST_PROJECT_ID, 'keep', postData);
  });
});
