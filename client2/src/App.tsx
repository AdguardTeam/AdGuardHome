import './main.pcss';
import './lib/ant/ant.less';
import React from 'react';
import ReactDOM from 'react-dom';
import Store, { storeValue } from 'Store';
import './lib/ant';

import App from './components/App';

const Container = () => {
    return (
        <Store.Provider value={storeValue}>
            <App/>
        </Store.Provider>
    );
};

ReactDOM.render(<Container />, document.getElementById('app'));
