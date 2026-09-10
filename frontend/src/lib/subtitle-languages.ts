// Shared catalog for Settings and collection Language menus. Analysis still
// supplies the per-video checklist; this list is the policy catalog.

export const COLLECTION_LANGUAGES: Array<{ code: string; name: string }> = [
  { code: 'en', name: 'English' },
  { code: 'es', name: 'Spanish' },
  { code: 'pt', name: 'Portuguese' },
  { code: 'fr', name: 'French' },
  { code: 'de', name: 'German' },
  { code: 'hi', name: 'Hindi' },
  { code: 'ja', name: 'Japanese' },
  { code: 'ko', name: 'Korean' },
  { code: 'zh', name: 'Chinese' },
  { code: 'ar', name: 'Arabic' },
  { code: 'it', name: 'Italian' },
  { code: 'ru', name: 'Russian' },
];

export function subtitleLanguageOffered(
  languages: Array<{ code: string; auto?: boolean }>,
  code: string,
  allowAuto = true,
): boolean {
  const want = code.trim().toLowerCase();
  if (!want) return false;
  return languages.some((language) => {
    if (language.auto && !allowAuto) return false;
    const have = language.code.trim().toLowerCase();
    return have === want || have.startsWith(`${want}-`) || have.startsWith(`${want}_`);
  });
}
