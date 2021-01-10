import T from './Translator';
import { Locale } from './locales';

export { messages, DatePickerLocale, Locale, DEFAULT_LOCALE, LANGUAGES, reactFormater } from './locales';
export type Translator = T<Locale>;
export default T;
