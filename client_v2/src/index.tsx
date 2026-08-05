import { render } from 'solid-js/web';

import App from './components/App';

import './index.pcss';

const root = document.getElementById('root')!;
root.innerHTML = '';

render(() => <App />, root);
