import * as fs from 'fs';
import {
    Project,
    VariableStatement,
    SyntaxKind,
    Node,
    Statement,
    ts,
    Identifier,
    SourceFile,
} from 'ts-morph';
import { 
    LOCALE_FOLDER_PATH,
    TRANSLATOR_CLASS_NAME,
    USE_INTL_NAME,
    trimQuotes,
} from '../consts';
import { checkForms, AvailableLocales } from '../../src/localization/Translator';

const project = new Project({
    tsConfigFilePath: './tsconfig.json',
});

let lang = 'ru';
let option = '';

if (process.argv.length > 2) { 
    lang = process.argv[2];
    option = process.argv[3];
}

const usedTranslations: string[] = [];
const usedPluralTranslations: string[] = [];

const problemFiles: string[] = [];
const sourceFiles = project.getSourceFiles();
const sourceFilesWithIntl = sourceFiles.filter((sf) => {
    return !!sf.getImportDeclarations().find((id) => {
        return !!id.getNamedImports().find((ni) => ni.getName() === USE_INTL_NAME)
    })
});
const getFileUsedIntl = (statements: Statement<ts.Statement>[]) => {
    statements.forEach((s) => {
        if (s instanceof VariableStatement) {
            s.forEachDescendant((node) => {
                let intVariableDeclaration: Identifier = null;
                switch (node.getKind()) {
                    case SyntaxKind.VariableDeclaration:
                        if (node.getSymbol()) {
                            const name = node.getSymbol().getName();
                            const callExp = node.getChildren().find((n) => n.getKind() === SyntaxKind.CallExpression);
                            if (callExp) {
                                const callExpIden = callExp.getChildren().find(n => n.getKind() === SyntaxKind.Identifier);
                                if (callExpIden && callExpIden.getSymbol().getName() === USE_INTL_NAME) {
                                    intVariableDeclaration = node as Identifier;
                                }
                            }
                        }
                        break;
                    default:
                        break;
                }
                if (intVariableDeclaration) {
                    intVariableDeclaration.findReferencesAsNodes().forEach((fr) => {
                        if (fr instanceof Node) {
                            const parent = fr.getParentIfKind(SyntaxKind.PropertyAccessExpression);
                            if (parent && (parent.getName() === 'getMessage' || parent.getName() === 'getPlural')) {
                                const syntaxList = parent.getNextSiblings().find((n) => n.getKind() === SyntaxKind.SyntaxList);
                                if (syntaxList) {
                                    const id = syntaxList.getChildren()[0];
                                    if (id && id.getKind() !== SyntaxKind.StringLiteral) {
                                        problemFiles.push(fr.getSourceFile().getFilePath());
                                    }
                                    if (id) {
                                        usedTranslations.push(trimQuotes(id.getText()));
                                        if (parent.getName() === 'getPlural') {
                                            usedPluralTranslations.push(trimQuotes(id.getText()));
                                        }
                                    }
                                }
                            }
                        }
                    })
                }
            });
        }
    })
}

const getFileUsedTranslations = (file: SourceFile) => {
    const namedImport = file.getImportDeclarations().find((id) => !!id.getNamedImports().find((ni) => ni.getName() === TRANSLATOR_CLASS_NAME));
    if (namedImport) {
        const identifier = namedImport.getImportClause().getNamedImports().find((iden) => iden.getName() === TRANSLATOR_CLASS_NAME);
        const translateReferences = identifier.getNodeProperty('name').findReferencesAsNodes();
        if (translateReferences.length > 0) {
            translateReferences.forEach((identifierNode) => {
                if (identifierNode.getParentIfKind(SyntaxKind.TypeReference)) {
                    const translatorVariable = identifierNode.getParent().getPreviousSibling().getPreviousSiblingIfKind(SyntaxKind.Identifier);
                    if (translatorVariable) {
                        translatorVariable.findReferencesAsNodes().forEach((node) => {
                            const parent = node.getParentIfKind(SyntaxKind.PropertyAccessExpression);
                            if (parent && (parent.getName() === 'getMessage' || parent.getName() === 'getPlural')) {
                                    
                                const syntaxList = parent.getNextSiblings().find((n) => n.getKind() === SyntaxKind.SyntaxList);
                                if (syntaxList) {
                                    const id = syntaxList.getChildren()[0];
                                    if (id && id.getKind() !== SyntaxKind.StringLiteral) {
                                        problemFiles.push(parent.getSourceFile().getFilePath());   
                                    }
                                    if (id) {
                                        usedTranslations.push(trimQuotes(id.getText()));
                                        if (parent.getName() === 'getPlural') {
                                            usedPluralTranslations.push(trimQuotes(id.getText()));
                                        }
                                    }
                                }
                            }
                        })
                    }
                }
            })
        }

    }
}
sourceFilesWithIntl.forEach((file) => {
    getFileUsedIntl(file.getStatements());
})

const sourceFilesWithTranslator = project.getSourceFiles().filter((sf) => {
    return !!sf.getImportDeclarations().find((id) => {
        return !!id.getNamedImports().find((ni) => ni.getName() === TRANSLATOR_CLASS_NAME)
    })
});
sourceFilesWithTranslator.forEach((file) => {
    getFileUsedTranslations(file);
})
const filteredUsedTranslations = Array.from(new Set(usedTranslations));
const filteredUsedPluralTranslations = Array.from(new Set(usedPluralTranslations));

if (problemFiles.length) {
    console.warn(`\n============== Files where translation id provided not as string ==============\n`);
    console.log(problemFiles.join('\n'));
    process.exit(255);
}

const allFiles = fs.readdirSync(LOCALE_FOLDER_PATH);
// Use ru or needed language
const translationFile = allFiles.find((file) => file.includes(`${lang}.json`));

if (!translationFile) {
    console.error('File not found');
    process.exit(255);
}

const translationsObject = JSON.parse(fs.readFileSync(`./src/lib/intl/__locales/${translationFile}`, { flag: 'r+' }) as unknown as string);
const translations = { 
    locale: translationFile,
    messages: Object.keys(translationsObject),
};

const someMessagesNotFound: string[] = [];
const notUsed: string[] = [];
const notFound: string[] = [];
const checkLocaleMessages = (locale: string, messages: string[]) => {
    filteredUsedTranslations.forEach(f => {
        if (!messages.includes(f)) {
            notFound.push(f);
        }
    });
    messages.forEach(t => {
        if (!filteredUsedTranslations.includes(t)) {
            notUsed.push(t);
        }
    });
    if (notFound.length > 0) {
        someMessagesNotFound.push(locale);
    }
}

const render = (data: string[], title: string) => {
    console.log(`============ ${title} ============`);
    console.table(data);
    console.log(`============ ${title} ============`);
}

checkLocaleMessages(translations.locale, translations.messages);

const checkPluralForm = () => {
    const pluralFormWrong: string[] = [];
    filteredUsedPluralTranslations.forEach((id) => {
        const message = translationsObject[id];
        if (!checkForms(message, lang as AvailableLocales, id)) {
            pluralFormWrong.push(id)
        }
    });
    return pluralFormWrong;
}

const plural = checkPluralForm();
if (!option && (someMessagesNotFound.length || plural.length > 0 )) {
    someMessagesNotFound.forEach(locale => console.error(`\nSome translatins for ${locale} was not found!\n`));
    plural.forEach(id => console.error(`\nTranslation with id: "${id}" - have wrong number of plural forms!\n`));
    process.exit(255);
}
if (option) {
    switch (option) {
        case '--show-missing': {
            render(notFound, 'NotFound')
            break;
        }
        case '--show-unused': {
            render(notUsed, 'notUsed')
            break;
        }
        case '--check-plurals': {
            render(plural, 'Wrong Plural Form')
        }
        default: {
            if (someMessagesNotFound.length) {
                someMessagesNotFound.forEach(locale => console.error(`\nSome translatins for ${locale} was not found!\n\n`)); 
                process.exit(255);
            }
        }
    } 
}
