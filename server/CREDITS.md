# CREDITS — 第三方素材与许可（server 后端）

> 宪法级约束：仅引入开源素材。本文件记录后端引入的素材及其许可。

## 字体（验证码渲染）

- **Go 字体（Go Mono Bold）** — `golang.org/x/image/font/gofont/gomonobold`
- 用途：图形验证码字符渲染（T-002b，替代此前不可读的伪随机像素块）。
- 来源：由 Bigelow & Holmes Inc. 专为 Go 项目制作（https://go.dev/blog/go-fonts）。
- 许可：**BSD-3-Clause**（与 Go 项目软件同一开源许可），属公认宽松开源许可，符合"仅开源素材"。
- 引入方式：经 Go module `golang.org/x/image` 依赖、构建期编译进二进制（等同 go:embed），无单独二进制文件需维护；许可随模块分发。
- 版权与许可全文见下。

```
Copyright (c) 2016 Bigelow & Holmes Inc.. All rights reserved.

Distribution of this font is governed by the following license. If you do not
agree to this license, including the disclaimer, do not distribute or modify
this font.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

    * Redistributions of source code must retain the above copyright notice,
      this list of conditions and the following disclaimer.

    * Redistributions in binary form must reproduce the above copyright notice,
      this list of conditions and the following disclaimer in the documentation
      and/or other materials provided with the distribution.

    * Neither the name of Google Inc. nor the names of its contributors may be
      used to endorse or promote products derived from this software without
      specific prior written permission.

DISCLAIMER: THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO,
THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
```

（x/image 库本身的 BSD-3-Clause 许可见 `go env GOMODCACHE`/golang.org/x/image@*/LICENSE。）
