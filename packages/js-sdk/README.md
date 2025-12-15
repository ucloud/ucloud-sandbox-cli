<p align="center">
  <img width="100" src="https://raw.githubusercontent.com/e2b-dev/E2B/refs/heads/main/readme-assets/logo-circle.png" alt="e2b logo">
</p>

<h4 align="center">  
  <a href="https://www.npmjs.com/package/e2b">
    <img alt="Last 1 month downloads for the JavaScript SDK" loading="lazy" width="200" height="20" decoding="async" data-nimg="1"
    style="color:transparent;width:auto;height:100%" src="https://img.shields.io/npm/dm/e2b?label=NPM%20Downloads">
  </a>
</h4>

<!---
<img width="100%" src="/readme-assets/preview.png" alt="Cover image">
--->
## What is E2B?
[E2B](https://sandbox.ucloudai.com/) is an open-source infrastructure that allows you to run AI-generated code in secure isolated sandboxes in the cloud. To start and control sandboxes, use our JavaScript SDK or Python SDK.

## Run your first Sandbox

### 1. Install SDK

```bash
npm i @e2b/code-interpreter
```

### 2. Get your E2B API key
1. Sign up to UCloud Sandbox [here](https://sandbox.ucloudai.com).
2. Get your API key [here](https://console.ucloud.cn/modelverse/experience/api-keys).
3. Set environment variable with your API key
```
E2B_API_KEY=e2b_***
```     

### 3. Execute code with code interpreter inside Sandbox

```ts
import { Sandbox } from '@e2b/code-interpreter'

const sbx = await Sandbox.create()
await sbx.runCode('x = 1')

const execution = await sbx.runCode('x+=1; x')
console.log(execution.text)  // outputs 2
```

### 4. Check docs
Visit [UCloud Sandbox documentation](https://sandbox.ucloudai.com/docs).

### 5. E2B cookbook
Visit our [Cookbook](https://github.com/e2b-dev/e2b-cookbook/tree/main) to get inspired by examples with different LLMs and AI frameworks.
