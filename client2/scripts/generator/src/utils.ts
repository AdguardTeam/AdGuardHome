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

export enum SchemaType {
    STRING = 'string',
    OBJECT = 'object',
    ARRAY = 'array',
    BOOLEAN = 'boolean',
    NUMBER = 'number',
    INTEGER = 'integer',
}

export interface Schema {
    allOf?: any[];
    example?: string;
    properties?: Record<string, Schema>;
    required?: string[];
    description?: string;
    enum?: string[];
    type: SchemaType;
    pattern?: string;
    oneOf?: any
    items?: Schema;
    additionalProperties?: Schema;
    $ref?: string;
    minItems?: number;
    maxItems?: number;
    maxLength?: number;
    minLength?: number;
    maximum?: number;
    minimum?: number;
}

export interface Parametr {
    description?: string;
    example?: string;
    in?: 'query' | 'body' | 'headers';
    name?: string;
    schema?: Schema;
    required?: boolean;
}

export interface RequestBody {
    content: {
        'application/json'?: {
            schema: Schema;
            example?: string;
        };
        'text/palin'?: {
            example?: string;
        }
    }
    required?: boolean;
}
export interface Response {
    content: {
        'application/json'?: {
            schema: Schema;
            example?: string;
        };
        'text/palin'?: {
            example?: string;
        }
    }
    description?: string;
}

export interface Schemas {
    parameters: Record<string, Parametr>;
    requestBodies: Record<string, RequestBody>;
    responses: Record<string, Response>;
    schemas: Record<string, Schema>;
}

export interface OpenApi {
    components: Schemas;
    paths: any;
}

/**
 * @param schemaProp: valueof shema.properties[key]
 * @param openApi: openapi object
 * @returns [propType - basicType or import one, isArray, isClass, isImport]
 */
interface SchemaParamParserReturn {
    type: string;
    isArray: boolean;
    isClass: boolean;
    isImport: boolean;
    isAdditional: boolean;
    isEnum: boolean;
}

const schemaParamParser = (schemaProp: Schema, openApi: OpenApi): SchemaParamParserReturn => {
    let type = '';
    let isImport = false;
    let isClass = false;
    let isArray = false;
    let isAdditional = false;
    let isEnum = false;

    if (schemaProp.$ref || schemaProp.additionalProperties?.$ref) {
        const temp = (schemaProp.$ref || schemaProp.additionalProperties?.$ref)!.split('/');

        if (schemaProp.additionalProperties) {
            isAdditional = true;
        }

        type = `${temp[temp.length - 1]}`;

        const cl = openApi.components.schemas[type];

        if (cl.$ref) {
            const link = schemaParamParser(cl, openApi);
            return {...link, type};
        }

        if (cl.type === 'string' && cl.enum) {
            isImport = true;
            isEnum = true;
        }

        if (cl.type === 'object' && !cl.oneOf) {
            isClass = true;
            isImport = true;
        } else if (cl.type === 'array') {
            const temp = schemaParamParser(cl.items!, openApi);
            type = temp.type;
            isArray = true;
            isClass = isClass || temp.isClass;
            isImport = isImport || temp.isImport;
            isEnum = isEnum || temp.isEnum;
        }
    } else if (schemaProp.type === 'array') {
        const temp = schemaParamParser(schemaProp.items!, openApi);
        type = temp.type
        isArray = true;
        isClass = isClass || temp.isClass;
        isImport = isImport || temp.isImport;
        isEnum = isEnum || temp.isEnum;
    } else {
        type = (TYPES as Record<any, string>)[schemaProp.type];
    }

    return { type, isArray, isClass, isImport, isAdditional, isEnum };
};

export { TYPES, toCamel, capitalize, uncapitalize, schemaParamParser };
