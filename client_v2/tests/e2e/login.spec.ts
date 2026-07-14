import { test } from '@playwright/test';

import { login } from '../helpers/login';


test.describe('Login', () => {
    test('should successfully log in with valid credentials', async ({ page }) => {
        await login(page);
    });
});
