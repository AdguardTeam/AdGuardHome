import { makeAutoObservable, observable } from 'mobx';

import Translator, { DEFAULT_LOCALE, messages, Locale, reactFormater } from 'Localization';
import { Store } from 'Store';

export default class UI {
    rootStore: Store;

    currentLang = DEFAULT_LOCALE;

    intl = new Translator<Locale>(Locale.en, messages, DEFAULT_LOCALE, reactFormater);

    constructor(rootStore: Store) {
        this.rootStore = rootStore;
        makeAutoObservable(this, {
            intl: observable.struct,
            rootStore: false,
        });
    }

    updateLang = (lang: Locale) => {
        this.currentLang = lang;
        this.intl = this.intl.updateTranslator(lang);
    };
}
