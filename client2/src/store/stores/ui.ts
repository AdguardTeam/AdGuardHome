import React from 'react';
import { makeAutoObservable, observable, action } from 'mobx';
import { translate } from '@adguard/translate';

import { Locale, DEFAULT_LOCALE, i18n } from 'Localization';
import { Store } from 'Store';
import { Store as InstallStore } from 'Store/installStore';

export default class UI {
    rootStore: Store | InstallStore;

    currentLang = DEFAULT_LOCALE;

    intl = translate.createReactTranslator<any>(i18n(this.currentLang), React);

    sidebarOpen = false;

    constructor(rootStore: Store | InstallStore) {
        this.rootStore = rootStore;
        makeAutoObservable(this, {
            intl: observable.struct,
            rootStore: false,
            sidebarOpen: observable,
            toggleSidebar: action,
        });
    }

    updateLang = (lang: Locale) => {
        this.currentLang = lang;
        this.intl = translate.createReactTranslator<any>(i18n(this.currentLang), React);
    };

    toggleSidebar = () => {
        this.sidebarOpen = !this.sidebarOpen;
    };
}
