#!/usr/bin/env node
// fork 独自の品質ゲート: en.json と ja.json のキー集合が完全一致することを検査する。
// 独自機能の i18n キーは en/ja のみに追加する規約 (.claude/rules/feature-development.md) のため、
// 片側だけの追加・本家マージでの解消ミスをここで検出する。
// 使い方: node .claude/scripts/check-i18n-parity.mjs [en.json ja.json]
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import path from 'node:path';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..');
const enPath = process.argv[2] ?? path.join(root, 'src/locales/en.json');
const jaPath = process.argv[3] ?? path.join(root, 'src/locales/ja.json');

const flatten = (obj, prefix = '') => Object.entries(obj).flatMap(([key, value]) =>
    value !== null && typeof value === 'object'
        ? flatten(value, `${prefix}${key}.`)
        : [`${prefix}${key}`]
);

const enKeys = new Set(flatten(JSON.parse(readFileSync(enPath, 'utf8'))));
const jaKeys = new Set(flatten(JSON.parse(readFileSync(jaPath, 'utf8'))));

const onlyEn = [...enKeys].filter(key => !jaKeys.has(key));
const onlyJa = [...jaKeys].filter(key => !enKeys.has(key));

if (onlyEn.length || onlyJa.length) {
    if (onlyEn.length) {
        console.error(`NG: en.json のみに存在するキー (${onlyEn.length} 件):`);
        onlyEn.forEach(key => console.error(`  ${key}`));
    }
    if (onlyJa.length) {
        console.error(`NG: ja.json のみに存在するキー (${onlyJa.length} 件):`);
        onlyJa.forEach(key => console.error(`  ${key}`));
    }
    process.exit(1);
}

console.log(`i18n parity OK (${enKeys.size} keys, en == ja)`);
