import translator from './lib/translator';
import { AllowedValues } from './lib/formatter';
import { getForm, GenericLocales, AvailableLocales } from './lib/plural';

type ExternalFormater = (data: any) => any;

class Translator<Locale extends GenericLocales[keyof GenericLocales], Formater = any> {
    private _currentLocale: Locale;

    private _formatter: ExternalFormater = (data: string[]) => data.join('');

    get currentLocale() {
        return this._currentLocale;
    }

    defaultLocale: Locale;

    updateTranslator = (locale: Locale) => {
        return new Translator<Locale, Formater>(
            this.defaultLocale,
            this.messages,
            locale,
            this._formatter,
        );
    };

    messages: Record<Locale, { [id: string]: string }>;

    constructor(
        defaultLocale: Locale,
        messages: Record<Locale, { [id: string]: string }>,
        currentLocale?: Locale,
        formatter?: ExternalFormater,
    ) {
        this.defaultLocale = defaultLocale;
        this._currentLocale = currentLocale ?? defaultLocale;
        this.messages = messages;

        if (formatter) {
            this._formatter = formatter;
        }
    }

    public getMessage(
        id: string,
        params: AllowedValues<Formater> = {},
    ): string {
        const str = this.messages[this._currentLocale][id]
            || this.messages[this.defaultLocale][id]
            || id;

        const tranlation = translator<Formater>(str, params);
        return this._formatter(tranlation);
    }

    public getPlural(
        id: string,
        number: number,
        params: AllowedValues<Formater> = {},
    ): string {
        let locale: Locale | null = null;
        if (this.messages[this._currentLocale][id]) {
            locale = this._currentLocale;
        } else if (this.messages[this.defaultLocale][id]) {
            locale = this.defaultLocale;
        }
        const str = this.messages[this._currentLocale][id]
            || this.messages[this.defaultLocale][id]
            || id;

        if (!locale) {
            throw new Error(`No translation for id: ${id}, neither in current locale: ${this._currentLocale} nor defaulkt locale ${this.defaultLocale}`);
        }

        return this._formatter(translator<Formater>(
            getForm(str, number, locale as AvailableLocales, id),
            { count: number, ...params },
        ));
    }
}

export default Translator;
