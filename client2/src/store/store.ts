import { createContext } from 'react';
import UI from './stores/ui';

export class Store {
    ui: UI;

    constructor() {
        this.ui = new UI(this);
    }
}

export const storeValue = new Store();

const StoreContext = createContext<Store>(storeValue);
export default StoreContext;
