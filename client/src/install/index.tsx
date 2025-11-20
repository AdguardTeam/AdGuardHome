import { createRoot } from 'react-dom/client';
import { Provider } from 'react-redux';

import '../components/App/index.css';
import '../components/ui/ReactTable.css';
import configureStore from '../configureStore';
import reducers from '../reducers/install';
import '../i18n';

import { Setup } from './Setup';
import { InstallState } from '../initialState';

const store = configureStore<InstallState>(reducers, {});

const container = document.getElementById('root');
const root = createRoot(container!);

root.render(
    <Provider store={store}>
        <Setup />
    </Provider>,
);
