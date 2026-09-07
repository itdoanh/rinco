import { test as base } from '@playwright/test';

type Context = {
  // Add custom context here
};

export const test = base.extend<Context>({});
export { expect } from '@playwright/test';
