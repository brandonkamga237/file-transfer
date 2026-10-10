#!/usr/bin/env node
// Publishes an article folder (article.md, meta.json, images/) to the LesCracks library.
// Usage: node docs/articles/publish.mjs docs/articles/partie-1 [--publish]
// Without --publish the article is saved as a draft, to review in the back office first.
// Running it again updates the article (by meta.id, else by title) instead of creating a second one.

import { readFile } from 'node:fs/promises';
import { basename, dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { createInterface } from 'node:readline';

process.on('uncaughtException', (error) => {
  console.error(error.message);
  process.exit(1);
});

const SITE =process.env.LESCRACKS_URL ?? 'https://lescracks.com';
const API = `${SITE}/api`;
const HERE = dirname(fileURLToPath(import.meta.url));
const PLATFORM = resolve(process.env.LESCRACKS_DIR ?? join(HERE, '../../../lescracks-platform'));
// The studio's own parser, so the stored blocks match what the back office would save.
const { markdownToBlocks } = await import(join(PLATFORM, 'frontend/src/lib/articleMarkdown.ts'));

const folder = resolve(process.argv[2] ?? '');
const status = process.argv.includes('--publish') ? 'PUBLISHED' : 'DRAFT';
const meta = JSON.parse(await readFile(join(folder, 'meta.json'), 'utf8'));
let markdown = await readFile(join(folder, 'article.md'), 'utf8');

function ask(question, hidden = false) {
  const rl = createInterface({ input: process.stdin, output: process.stdout, terminal: true });
  if (hidden) rl._writeToOutput = (text) => { if (!text.includes(question)) return; rl.output.write(text); };
  return new Promise((done) => rl.question(question, (answer) => { rl.close(); if (hidden) process.stdout.write('\n'); done(answer); }));
}

let cookie = '';
async function call(path, init = {}) {
  const response = await fetch(`${API}${path}`, { ...init, headers: { ...init.headers, cookie } });
  const text = await response.text();
  if (!response.ok) throw new Error(`${init.method ?? 'GET'} ${path} → ${response.status} ${text}`);
  return text ? JSON.parse(text) : null;
}

async function imageFile(path) {
  const type = path.endsWith('.png') ? 'image/png' : 'image/jpeg';
  return new File([await readFile(path)], basename(path), { type });
}

const username = process.env.LESCRACKS_ADMIN_USER ?? await ask('Identifiant admin : ');
const password = process.env.LESCRACKS_ADMIN_PASSWORD ?? await ask('Mot de passe : ', true);
const login = await fetch(`${API}/admin/auth/login`, {
  method: 'POST',
  headers: { 'content-type': 'application/json' },
  body: JSON.stringify({ username, password }),
});
if (!login.ok) throw new Error(`Connexion refusée (${login.status}).`);
cookie = login.headers.getSetCookie().map((line) => line.split(';')[0]).join('; ');

const categories = await call('/categories');
const category = categories.find((item) => item.name === meta.category);
if (!category) throw new Error(`Catégorie introuvable : ${meta.category}`);
const tags = await call('/tags');
const tagIds = meta.tags.map((name) => {
  const tag = tags.find((item) => item.name === name && item.categoryId === category.id) ?? tags.find((item) => item.name === name);
  if (!tag) throw new Error(`Tag introuvable : ${name}`);
  return tag.id;
});

// Figures are stored on the platform, so the article never depends on GitHub being reachable.
const images = [...new Set([...markdown.matchAll(/!\[[^\]]*\]\((\S+?)[\s)]/g)].map((match) => match[1]))];
for (const url of images) {
  const local = join(folder, 'images', basename(new URL(url, SITE).pathname));
  const form = new FormData();
  form.append('image', await imageFile(local));
  const uploaded = await call('/admin/upload/image', { method: 'POST', body: form });
  markdown = markdown.replaceAll(url, uploaded.url);
  console.log(`image ${basename(local)} → ${uploaded.url}`);
}

// meta.id pins the article once published, so a later title change updates it instead of duplicating it.
const existing = meta.id
  ? { id: meta.id }
  : (await call(`/resources/admin?${new URLSearchParams({ search: meta.title, kind: 'ARTICLE', size: '50' })}`))
    .content.find((item) => item.title === meta.title);
const data = {
  title: meta.title,
  description: meta.description,
  coverImage: existing?.coverImage ?? '',
  categoryId: category.id,
  tagIds,
  status,
  scheduledAt: null,
  body: markdownToBlocks(markdown),
};
const form = new FormData();
form.append('data', new Blob([JSON.stringify(data)], { type: 'application/json' }));
form.append('coverImageFile', await imageFile(join(folder, meta.cover)));

const saved = existing
  ? await call(`/resources/admin/articles/${existing.id}`, { method: 'PUT', body: form })
  : await call('/resources/admin/articles', { method: 'POST', body: form });

await fetch(`${API}/admin/auth/logout`, { method: 'POST', headers: { cookie } });
console.log(`${existing ? 'Mis à jour' : 'Créé'} (${status === 'PUBLISHED' ? 'publié' : 'brouillon'}) : ${SITE}/ressources/${saved.slug ?? saved.id}`);
console.log(`Back office : ${SITE}/admin/articles/${saved.id}`);
