export const FILTERED_FINDINGS_LABEL = 'Filtered findings';
export const REFILTER_SELECTED_LABEL = 'Re-filter selected';
export const RECHECK_SELECTED_LABEL = 'Re-check selected';
export const RESTORE_VISIBILITY_LABEL = 'Restore visibility';
export const RESTORE_AND_TRUST_LABEL = 'Restore and trust…';
export const FILTERED_EMPTY_TITLE = 'No filtered findings yet';
export const FILTERED_EMPTY_BODY = 'Scout keeps filtered findings here for owner review. They stay hidden from review queues, reports, drafts, and Google result lists until you restore visibility.';

export function reasonLabel(reason) {
  const labels = {
    spam: 'Spam or promotion',
    bot_like: 'Bot-like activity',
    low_quality_account: 'Low-quality account',
    ai_generated: 'Likely AI-generated',
    trusted_override: 'Trusted override',
  };
  return labels[reason] || reason || 'Filtered';
}
