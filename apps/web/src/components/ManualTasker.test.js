// @vitest-environment jsdom
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';
import ManualTasker from './ManualTasker.svelte';
import { manualScout, manualScoutCommit, posts } from '../lib/api.js';

vi.mock('../lib/api.js', () => ({
  manualScout: vi.fn(),
  manualScoutCommit: vi.fn(),
  posts: {
    get: vi.fn(),
  },
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
    manualScout.mockResolvedValue({ status: 'saved', post_id: 't3_test', post: { title: 'Test Post', url: 'https://reddit.com/r/test' } });
    posts.get.mockResolvedValue({ title: 'Test Post', url: 'https://reddit.com/r/test', platform: 'reddit' });

    render(ManualTasker, { props: { projectId: TEST_PROJECT_ID } });

    const urlInput = screen.getByLabelText('URL');
    await fireEvent.input(urlInput, { target: { value: 'https://reddit.com/r/test' } });

    const submitBtn = screen.getByRole('button', { name: /scout url/i });
    await fireEvent.click(submitBtn);

    expect(manualScout).toHaveBeenCalledTimes(1);
    expect(manualScout).toHaveBeenCalledWith(TEST_PROJECT_ID, 'https://reddit.com/r/test', 'reddit');
    expect(posts.get).toHaveBeenCalledWith(TEST_PROJECT_ID, 't3_test');
  });

  it('shows loading state during submit', async () => {
    let resolveScout;
    const scoutPromise = new Promise((res) => { resolveScout = res; });
    manualScout.mockReturnValue(scoutPromise);

    render(ManualTasker, { props: { projectId: TEST_PROJECT_ID } });

    const urlInput = screen.getByLabelText('URL');
    await fireEvent.input(urlInput, { target: { value: 'https://reddit.com/r/test' } });

    const submitBtn = screen.getByRole('button', { name: /scout url/i });
    await fireEvent.click(submitBtn);

    expect(screen.getByText(/scouting.../i)).toBeTruthy();
    expect(screen.queryByText(/scout url/i)).toBeNull();

    posts.get.mockResolvedValue({ title: 'Test', platform: 'reddit' });
    resolveScout({ status: 'saved', post_id: 't3_loading', post: { title: 'Test' } });
    await waitFor(() => expect(screen.queryByText(/scouting.../i)).toBeNull());
  });

  it('shows "saved" success state with green display', async () => {
    manualScout.mockResolvedValue({
      status: 'saved',
      post_id: 't3_save',
      post: { title: 'Great Post', url: 'https://reddit.com/r/test', final_score: 7.5 },
      score: 7.5,
    });
    posts.get.mockResolvedValue({
      title: 'Great Post', url: 'https://reddit.com/r/test', final_score: 7.5,
      platform: 'reddit', status: 'new', body: 'Great content',
    });

    render(ManualTasker, { props: { projectId: TEST_PROJECT_ID } });

    const urlInput = screen.getByLabelText('URL');
    await fireEvent.input(urlInput, { target: { value: 'https://reddit.com/r/test' } });

    const submitBtn = screen.getByRole('button', { name: /scout url/i });
    await fireEvent.click(submitBtn);

    await waitFor(() => expect(screen.getByText('Saved')).toBeTruthy());
    expect(posts.get).toHaveBeenCalledWith(TEST_PROJECT_ID, 't3_save');
    await waitFor(() => expect(screen.getByText('Great Post')).toBeTruthy());
  });

  it('shows "already_scouted" state with blue info display', async () => {
    manualScout.mockResolvedValue({
      status: 'already_scouted',
      post_id: 't3_old',
      post: { title: 'Old Post', url: 'https://reddit.com/r/test', final_score: 5.0 },
    });
    posts.get.mockResolvedValue({
      title: 'Old Post', url: 'https://reddit.com/r/test', final_score: 5.0,
      platform: 'reddit', status: 'reviewed',
    });

    render(ManualTasker, { props: { projectId: TEST_PROJECT_ID } });

    const urlInput = screen.getByLabelText('URL');
    await fireEvent.input(urlInput, { target: { value: 'https://reddit.com/r/test' } });

    const submitBtn = screen.getByRole('button', { name: /scout url/i });
    await fireEvent.click(submitBtn);

    await waitFor(() => expect(screen.getByText('Already Scouted')).toBeTruthy());
    expect(posts.get).toHaveBeenCalledWith(TEST_PROJECT_ID, 't3_old');
    await waitFor(() => expect(screen.getByText('Old Post')).toBeTruthy());
  });

  it('shows "needs_decision" with Keep/Exclude buttons and post data', async () => {
    manualScout.mockResolvedValue({
      status: 'needs_decision',
      post_id: 't3_decide',
      post: { title: 'Borderline Post', url: 'https://reddit.com/r/test', post_score: 1.5 },
      score: 1.5,
    });
    posts.get.mockResolvedValue({
      title: 'Borderline Post', url: 'https://reddit.com/r/test', post_score: 1.5,
      platform: 'reddit', status: 'drafted',
    });

    render(ManualTasker, { props: { projectId: TEST_PROJECT_ID } });

    const urlInput = screen.getByLabelText('URL');
    await fireEvent.input(urlInput, { target: { value: 'https://reddit.com/r/test' } });

    const submitBtn = screen.getByRole('button', { name: /scout url/i });
    await fireEvent.click(submitBtn);

    await waitFor(() => expect(screen.getByText('Needs Decision')).toBeTruthy());
    expect(posts.get).toHaveBeenCalledWith(TEST_PROJECT_ID, 't3_decide');
    await waitFor(() => expect(screen.getByText('Borderline Post')).toBeTruthy());
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
      post_id: 't3_commit',
      post: postData,
      score: 1.5,
    });
    posts.get.mockResolvedValueOnce({
      title: 'Borderline Post', url: 'https://reddit.com/r/test', post_score: 1.5,
      platform: 'reddit', status: 'drafted',
    });
    manualScoutCommit.mockResolvedValue({ status: 'saved', post: postData });
    posts.get.mockResolvedValueOnce({
      title: 'Borderline Post', url: 'https://reddit.com/r/test', post_score: 1.5,
      platform: 'reddit', status: 'drafted',
    });

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
