// eslint-disable-next-line import/no-relative-packages
import twosky from 'Twosky';

const homeV2 = twosky.find((p) => p.project_id === 'home_v2');

export const LANGUAGES: Record<string, string> = homeV2?.languages ?? {};
export const LANGUAGE_NAMES: Record<string, string> = homeV2?.languages ?? {};
export const BASE_LOCALE = homeV2?.base_locale ?? 'en';
