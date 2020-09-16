const path = require('path');
const twoskyConfig = require('../../.twosky.json')[0];

const {languages} = twoskyConfig;
const LOCALES_DIR = '../../client/src/__locales';
const LOCALES_LIST = Object.keys(languages);
const BASE_FILE = 'en.json';

const main = () => {
    const pathToBaseFile = path.join(LOCALES_DIR, BASE_FILE);
    const baseLanguageJson = require(pathToBaseFile);

    const summary = {};

    LOCALES_LIST.forEach((locale) => {
        const pathToFile = path.join(LOCALES_DIR, `${locale}.json`);
        if (pathToFile === pathToBaseFile) {
            return;
        }

        let total = 0;
        let translated = 0;

        const languageJson = require(pathToFile);
        for (let key in baseLanguageJson) {
            total += 1;
            if (key in languageJson) {
                translated += 1;
            }
        }

        summary[locale] = Math.round(translated / total * 10000) / 100;
    });

    console.log('Translations summary:');
    for (let key in summary) {
        console.log(`${key}, translated ${summary[key]}%`);
    }
}

main();
