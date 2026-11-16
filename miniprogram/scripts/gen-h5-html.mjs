// 构建后生成 dist/h5/index.html：
// Taro 4.3 不产出 index.html，手写模板无法预知 webpack 的 initial chunk 文件名
// （app.js 之外的 js/*.js 都是 initial chunk，缺一个就是静默白屏：
//   runtime 会一直等待该 chunk，React 不挂载、不报错）。
// 因此以 h5/index.html 为模板，把 <!--STYLES--> / <!--SCRIPTS--> 占位符
// 替换为对 dist/h5/css、dist/h5/js 的实际扫描结果。
import { readFileSync, writeFileSync, readdirSync, existsSync } from 'node:fs'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')
const dist = join(root, 'dist/h5')
const template = join(root, 'h5/index.html')

if (!existsSync(template)) {
  console.error('缺少模板 h5/index.html')
  process.exit(1)
}

// initial chunk：js/ 目录下所有 .js（排除 sourcemap / LICENSE），app.js 必须最后执行
const jsFiles = readdirSync(join(dist, 'js'))
  .filter((f) => f.endsWith('.js'))
  .sort((a, b) => (a === 'app.js' ? 1 : b === 'app.js' ? -1 : a.localeCompare(b, undefined, { numeric: true })))
const cssFiles = readdirSync(join(dist, 'css')).filter((f) => f.endsWith('.css'))

if (!jsFiles.includes('app.js')) {
  console.error('未找到 dist/h5/js/app.js，构建可能失败')
  process.exit(1)
}

const styles = cssFiles.map((f) => `  <link rel="stylesheet" href="/acv-h5/css/${f}" />`).join('\n')
const scripts = jsFiles.map((f) => `  <script src="/acv-h5/js/${f}"></script>`).join('\n')

let html = readFileSync(template, 'utf8')
html = html.replace('<!--STYLES-->', styles).replace('<!--SCRIPTS-->', scripts)

writeFileSync(join(dist, 'index.html'), html)
console.log(`index.html 已生成：${jsFiles.length} 个 initial chunk（${jsFiles.join(', ')}），${cssFiles.length} 个样式`)
