import { createContext } from 'react';
import Install from './stores/Install';
import UI from './stores/ui';

export class Store {
    ui: UI;

    install: Install;

    constructor() {
        this.ui = new UI(this);
        this.install = new Install(this);
    }
}

export const storeValue = new Store();

const StoreContext = createContext<Store>(storeValue);
export default StoreContext;
