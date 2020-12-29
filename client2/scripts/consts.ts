export const OPEN_API_PATH = '../openapi/openapi.yaml';
export const ENT_DIR = './src/lib/entities';
export const API_DIR = './src/lib/apis';
export const LOCALE_FOLDER_PATH = './src/lib/intl/__locales';
export const TRANSLATOR_CLASS_NAME = 'Translator';
export const USE_INTL_NAME = 'useIntl';

export const trimQuotes = (str: string) => {
    return str.replace(/\'|\"/g, '');
};

export const GENERATOR_ENTITY_ALLIAS = 'Entities/';