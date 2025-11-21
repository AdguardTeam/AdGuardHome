import { createRoot } from 'react-dom/client';
import { Provider } from 'react-redux';

import '../components/App/index.css';
import '../components/ui/ReactTable.css';
import configureStore from '@/configureStore';
import reducers from '@/reducers/login';
import '../i18n';

import { LoginState } from '@/initialState';
import { Login } from './Login';

const store = configureStore<LoginState>(reducers, {});

const container = document.getElementById('root');
const root = createRoot(container!);

root.render(
    <Provider store={store}>
        <Login />
    </Provider>,
);
