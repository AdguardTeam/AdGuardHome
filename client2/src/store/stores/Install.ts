import InstallApi from 'Apis/install';
import AddressesInfoBeta, { IAddressesInfoBeta } from 'Entities/AddressesInfoBeta';
import { ICheckConfigRequestBeta } from 'Entities/CheckConfigRequestBeta';
import CheckConfigResponse, { ICheckConfigResponse } from 'Entities/CheckConfigResponse';
import { IInitialConfigurationBeta } from 'Entities/InitialConfigurationBeta';
import { errorChecker } from 'Helpers/apiErrors';
import { flow, makeAutoObservable } from 'mobx';

import { Store } from 'Store/installStore';

export default class Install {
    rootStore: Store;

    addresses: AddressesInfoBeta | null;

    constructor(rootStore: Store) {
        this.rootStore = rootStore;
        this.addresses = null;

        makeAutoObservable(this, {
            rootStore: false,
            getAddresses: flow,
        });
        this.getAddresses();
    }

    * getAddresses() {
        const response = yield InstallApi.installGetAddressesBeta();
        const { result } = errorChecker<IAddressesInfoBeta>(response);
        if (result) {
            this.addresses = new AddressesInfoBeta(result);
        }
    }

    static async checkConfig(config: ICheckConfigRequestBeta) {
        const response = await InstallApi.installCheckConfigBeta(config);
        const { result } = errorChecker<ICheckConfigResponse>(response);
        if (result) {
            return new CheckConfigResponse(result);
        }
    }

    static async configure(config: IInitialConfigurationBeta) {
        const response = await InstallApi.installConfigureBeta(config);
        const { result } = errorChecker<number>(response);
        if (result) {
            return true;
        }
    }
}
