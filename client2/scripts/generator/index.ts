import * as fs from 'fs';
import * as YAML from 'yaml';
import { OPEN_API_PATH } from '../consts';

import EntitiesGenerator from './src/generateEntities';
import ApisGenerator from './src/generateApis';
import { OpenApi } from './src/utils';


const generateApi = (openApi: OpenApi) => {
    const ent = new EntitiesGenerator(openApi);
    ent.save();

    // const api = new ApisGenerator(openApi);
    // api.save();
}

const openApiFile = fs.readFileSync('./scripts/generator/v1.yaml', 'utf8');
generateApi(YAML.parse(openApiFile));
