const toCamel = (s: string) => {
    return s.replace(/([-_][a-z])/ig, ($1) => {
        return $1.toUpperCase()
            .replace('-', '')
            .replace('_', '');
    });
};
const capitalize = (s: string) => {
    return s[0].toUpperCase() + s.slice(1);
};
const uncapitalize = (s: string) => {
    return s[0].toLowerCase() + s.slice(1);
};
const TYPES = {
    integer: 'number',
    float: 'number',
    number: 'number',
    string: 'string',
    boolean: 'boolean',
};

/**
 * @param schemaProp: valueof shema.properties[key]
 * @param openApi: openapi object
 * @returns [propType - basicType or import one, isArray, isClass, isImport]
 */
const schemaParamParser = (schemaProp: any, openApi: any): [string, boolean, boolean, boolean, boolean] => {
    let type = '';
    let isImport = false;
    let isClass = false;
    let isArray = false;
    let isAdditional = false;

    if (schemaProp.$ref || schemaProp.additionalProperties?.$ref) {
        const temp = (schemaProp.$ref || schemaProp.additionalProperties?.$ref).split('/');

        if (schemaProp.additionalProperties) {
            isAdditional = true;
        }

        type = `${temp[temp.length - 1]}`;

        const cl = openApi ? openApi.components.schemas[type] : {};

        if (cl.$ref) {
            const link = schemaParamParser(cl, openApi);
            link.shift();
            return [type, ...link] as any;
        }

        if (cl.type === 'string' && cl.enum) {
            isImport = true;
        }

        if (cl.type === 'object' && !cl.oneOf) {
            isClass = true;
            isImport = true;
        } else if (cl.type === 'array') {
            const temp: any = schemaParamParser(cl.items, openApi);
            type = `${temp[0]}`;
            isArray = true;
            isClass = isClass || temp[2];
            isImport = isImport || temp[3];
        }
    } else if (schemaProp.type === 'array') {
        const temp: any = schemaParamParser(schemaProp.items, openApi);
        type = `${temp[0]}`;
        isArray = true;
        isClass = isClass || temp[2];
        isImport = isImport || temp[3];
    } else {
        type = (TYPES as Record<any, string>)[schemaProp.type];
    }
    if (!type) {
        // TODO: Fix bug with Error fields.
        type = 'any';
        // throw new Error('Failed to find entity type');
    }

    return [type, isArray, isClass, isImport, isAdditional];
};

export { TYPES, toCamel, capitalize, uncapitalize, schemaParamParser };
