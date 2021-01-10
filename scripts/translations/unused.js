const fs = require('fs');
const path = require('path');

const SRC_DIR = '../../client/src/'
const LOCALES_DIR = '../../client/src/__locales';
const BASE_FILE = path.join(LOCALES_DIR, 'en.json');

// Strings that may be found by the algorithm,
// but in fact they are used.
const KNOWN_USED_STRINGS = {
    'blocking_mode_refused': true,
    'blocking_mode_nxdomain': true,
    'blocking_mode_custom_ip': true,
}

function traverseDir(dir, callback) {
    fs.readdirSync(dir).forEach(file => {
        let fullPath = path.join(dir, file);
        if (fs.lstatSync(fullPath).isDirectory()) {
            traverseDir(fullPath, callback);
        } else {
            callback(fullPath);
        }
    });
}

const contains = (key, files) => {
    for (let file of files) {
        if (file.includes(key)) {
            return true;
        }
    }

    return false;
}

const main = () => {
    const baseLanguage = require(BASE_FILE);
    const files = [];

    traverseDir(SRC_DIR, (path) => {
        const canContain = (path.endsWith('.js') || path.endsWith('.json')) &&
            !path.includes(LOCALES_DIR);

        if (canContain) {
            files.push(fs.readFileSync(path).toString());
        }
    });

    const unused = [];
    for (let key in baseLanguage) {
        if (!contains(key, files) && !KNOWN_USED_STRINGS[key]) {
            unused.push(key);
        }
    }

    console.log('Unused keys:');
    for (let key of unused) {
        console.log(key);
    }
}

main();
