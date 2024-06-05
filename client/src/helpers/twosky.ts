// eslint-disable-next-line import/no-relative-packages
import twosky from '../../../.twosky.json';

console.log(twosky[0]);

export const { languages: LANGUAGES, base_locale: BASE_LOCALE } = twosky[0];
