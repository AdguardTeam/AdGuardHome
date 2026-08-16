import { render } from 'solid-js/web';

import '../index.pcss';

import { ForgotPassword } from './ForgotPassword';

const root = document.getElementById('root')!;

render(() => <ForgotPassword />, root);
