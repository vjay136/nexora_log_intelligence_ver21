import {test,expect} from '@playwright/test';

test('R8 real provider form verifies two NVIDIA-compatible profiles, JSON/SSE vision and incomplete acknowledgements',async({page})=>{
  test.skip(!process.env.NLI_LOCAL_AI_PROOF,'Requires the isolated local controller and synthetic provider fixture');
  test.setTimeout(120_000);expect(new URL(process.env.NLI_BASE_URL||'http://127.0.0.1:12383').hostname).toMatch(/^(127\.0\.0\.1|localhost)$/);
  await page.goto('/');
  await page.locator('#login-form [name=username]').fill('admin');
  await page.locator('#login-form [name=password]').fill(process.env.NLI_ADMIN_PASSWORD);
  await page.locator('#login-form button[type=submit]').click();
  await expect(page.locator('#app-view')).toBeVisible();
  await page.getByTestId('nav-ai').click();
  const fakeURL='http://127.0.0.1:18443/v1';
  const form=page.locator('#ai-provider-form');
  async function configure(id,model,endpoint){
    await page.locator('[data-action=ai-profile-new]').click();
    await form.locator('[name=id]').fill(id);
    await form.locator('[name=name]').fill(id);
    await form.locator('[name=provider]').selectOption('nvidia_nim');
    await form.locator('[name=model]').fill(model);
    await form.locator('[name=endpoint]').fill(endpoint);
    await form.locator('[name=credential_source]').selectOption('env_ref');
    await form.locator('[name=credential_ref]').fill('NVIDIA_API_KEY');
    await form.locator('[name=enabled]').check();
    const saved=page.waitForResponse(response=>new URL(response.url()).pathname==='/api/v1/admin/ai-provider'&&response.request().method()==='PUT');
    await form.getByRole('button',{name:'Save provider settings',exact:true}).click();
    expect((await saved).status()).toBe(200);
    await expect(form.locator('[name=id]')).toHaveValue(id);
    await expect(form.locator('[name=api_key]')).toHaveValue('');
  }
  async function probe(){
    const result=page.waitForResponse(response=>new URL(response.url()).pathname==='/api/v1/admin/ai-provider/test'&&response.request().method()==='POST');
    await form.getByRole('button',{name:'Test connection',exact:true}).click();
    const response=await result;
    expect(response.status()).toBe(200);
    const body=await response.json();
    expect(JSON.stringify(body)).not.toContain('nli-r8-local-fake-provider-key');
    await expect(page.locator('#ai-provider-test-status')).not.toBeEmpty();
    return body;
  }
  await configure('nvidia-text-local-proof','deepseek-ai/deepseek-v4.1-flash',fakeURL);
  expect(await probe()).toMatchObject({connected:true});
  await configure('nvidia-vision-local-proof','minimaxai/minimax-m3.1-flash',fakeURL+'/chat/completions');
  await form.getByText('Explicit connection-test options',{exact:true}).click();
  await form.locator('[name=test_mode]').selectOption('stream');
  await form.locator('[name=test_image_url]').fill('https://assets.ngc.nvidia.com/products/api-catalog/phi-3-5-vision/example1b.jpg');
  expect(await probe()).toMatchObject({connected:true,acknowledged:true});
  await expect(form.locator('[name=model]')).toHaveValue('minimaxai/minimax-m3.1-flash');
  await form.locator('[name=model]').fill('simulate-reasoning');
  const incomplete=await probe();
  expect(incomplete.connected).toBe(false);
  expect(incomplete.acknowledged).toBe(true);expect(incomplete.tested_config).toMatchObject({configuration:'unsaved_draft',model:'simulate-reasoning'});
  await expect(page.locator('#ai-provider-test-status')).toContainText(/incomplete|budget|reasoning/i);
  await form.locator('[name=credential_source]').selectOption('encrypted_inline');
  await form.locator('[name=api_key]').fill('invalid-local-probe-key');
  const rejected=await probe();
  expect(rejected.connected).toBe(false);
  await expect(page.locator('#ai-provider-test-status')).toContainText(/auth|credential|401/i);
  await page.locator('#ai-profile-choice').selectOption('nvidia-vision-local-proof');
  await page.locator('[data-action=ai-profile-load]').click();
  await expect(form.locator('[name=model]')).toHaveValue('minimaxai/minimax-m3.1-flash');
  await expect(form.locator('[name=api_key]')).toHaveValue('');
  expect(await page.locator('#ai-profile-choice option').count()).toBeGreaterThanOrEqual(3);
  await page.screenshot({path:test.info().outputPath('r8-ai-connection-profiles.png'),fullPage:true});
  page.once('dialog',dialog=>{expect(dialog.message()).toContain('nvidia-vision-local-proof');return dialog.accept();});
  const cleared=page.waitForResponse(r=>r.request().method()==='POST'&&r.url().includes('action=clear-credential'));
  await form.getByRole('button',{name:'Remove credential',exact:true}).click();
  expect((await cleared).status()).toBe(200);
  await expect(form.locator('[name=enabled]')).not.toBeChecked();
  await expect(form.locator('[name=api_key]')).toHaveValue('');
  await expect(page.locator('#ai-profile-choice')).toContainText('nvidia-text-local-proof');
  page.once('dialog',dialog=>dialog.accept());
  const deleted=page.waitForResponse(r=>r.request().method()==='DELETE'&&r.url().includes('id=nvidia-vision-local-proof'));
  await form.getByRole('button',{name:'Delete profile',exact:true}).click();
  expect((await deleted).status()).toBe(200);
  await expect(page.locator('#ai-profile-choice')).not.toContainText('nvidia-vision-local-proof');
  await expect(page.locator('#ai-profile-choice')).toContainText('nvidia-text-local-proof');

});

test('R8 real configured AI returns a separate native mapping proposal and requires explicit import',async({page})=>{
 test.skip(!process.env.NLI_LOCAL_AI_PROOF,'Requires isolated controller and synthetic provider fixture');test.setTimeout(120_000);expect(new URL(process.env.NLI_BASE_URL||'http://127.0.0.1:12383').hostname).toMatch(/^(127\.0\.0\.1|localhost)$/);
 await page.goto('/');await page.locator('#login-form [name=username]').fill('admin');await page.locator('#login-form [name=password]').fill(process.env.NLI_ADMIN_PASSWORD);await page.locator('#login-form button[type=submit]').click();await expect(page.locator('#app-view')).toBeVisible();await page.getByTestId('nav-ai').click();await page.locator('[data-action=ai-profile-new]').click();const form=page.locator('#ai-provider-form');
 for(const [name,value] of Object.entries({id:'nvidia-mapping-local-proof',name:'Synthetic native mapping proof',model:'simulate-mapping',endpoint:'http://127.0.0.1:18443/v1',priority:'0',fallback_group:'mapping-proof',credential_ref:'NVIDIA_API_KEY'}))await form.locator(`[name=${name}]`).fill(value);await form.locator('[name=provider]').selectOption('nvidia_nim');await form.locator('[name=credential_source]').selectOption('env_ref');await form.locator('[name=enabled]').check();const saved=page.waitForResponse(r=>r.request().method()==='PUT'&&new URL(r.url()).pathname==='/api/v1/admin/ai-provider');await form.getByRole('button',{name:'Save provider settings',exact:true}).click();expect((await saved).status()).toBe(200);
 await page.getByTestId('nav-field-mapping').click();const raw='20250811 080053105+0100 aaa-gmailrgi009 smtp 3692 0 140026854221568 Note;SieveRuleMsg status=failed:reason=notneeded:server=cloud';await page.locator('#nfi-raw').fill(raw);await page.getByRole('button',{name:'Add Custom Field',exact:true}).click();await page.locator('[data-qrw-key=name]').fill('operator_manual_field');const host=page.locator('[data-ai-mapping-host]');const suggested=page.waitForResponse(r=>new URL(r.url()).pathname==='/api/v1/field-mapping/ai-suggest');await host.getByRole('button',{name:'Suggest with AI',exact:true}).click();const response=await suggested,body=await response.json();expect(response.status()).toBe(200);expect(body).toMatchObject({review_only:true,persisted:false,activated:false,validation:{syntax_valid:true,extraction_valid:true,ready_for_publish:false}});await expect(page.locator('[data-qrw-key=name]')).toHaveValue('operator_manual_field');await expect(host.locator('[data-ai-mapping-json]')).toContainText('SieveRuleMsg');await page.screenshot({path:test.info().outputPath('r8-ai-separate-mapping-proposal.png'),fullPage:true});
 page.once('dialog',dialog=>dialog.accept());await host.getByRole('button',{name:'Import proposed mapping',exact:true}).click();await expect(page.locator('[data-qrw-key=name]')).toHaveCount(5);for(const control of await page.locator('[data-qrw-key=confirmed]').all()){await expect(control).not.toBeChecked();await control.check();}const verified=page.waitForResponse(r=>new URL(r.url()).pathname==='/api/v1/assurance/mapping-preview');await page.getByRole('button',{name:'Verify on controller',exact:true}).click();expect((await verified).status()).toBe(200);await expect(page.locator('#nfi-controller-result')).toContainText('SieveRuleMsg');
 await page.getByTestId('nav-ai').click();await page.locator('#ai-profile-choice').selectOption('nvidia-mapping-local-proof');await page.locator('[data-action=ai-profile-load]').click();page.once('dialog',dialog=>dialog.accept());const deleted=page.waitForResponse(r=>r.request().method()==='DELETE'&&r.url().includes('id=nvidia-mapping-local-proof'));await page.getByRole('button',{name:'Delete profile',exact:true}).click();expect((await deleted).status()).toBe(200);
});
