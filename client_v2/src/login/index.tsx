import { render } from 'solid-js/web';

import '../index.pcss';

import { Login } from './Login';

const root = document.getElementById('root')!;

render(() => <Login />, root);
