import * as fs from 'fs';
import * as YAML from 'yaml';
import { OPEN_API_PATH } from '../consts';

import EntitiesGenerator from './src/generateEntities';
import ApisGenerator from './src/generateApis';


const generateApi = (openApi: Record<string, any>) => {
    const ent = new EntitiesGenerator(openApi);
    ent.save();

    const api = new ApisGenerator(openApi);
    api.save();
}

const openApiFile = fs.readFileSync(OPEN_API_PATH, 'utf8');
generateApi(YAML.parse(openApiFile));
