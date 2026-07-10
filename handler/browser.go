package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// BrowserPageHandler 显示浏览器不兼容提示页
// 对应 PHP: route/browser.php (非 download 分支)
// 当用户使用老旧浏览器访问论坛时，显示升级浏览器提示
func BrowserPageHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(browserPageHTML)
	}
}

// BrowserDownloadHandler 浏览器下载重定向
// 对应 PHP: route/browser.php (download 分支)
// GET /browser-download/{type} -> 302 跳转到官方下载页
func BrowserDownloadHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		browserType := chi.URLParam(r, "type")

		var redirectURL string
		switch browserType {
		case "chrome":
			redirectURL = "https://www.google.com/chrome/"
		case "firefox":
			redirectURL = "https://www.mozilla.org/firefox/new/"
		case "ie":
			redirectURL = "https://www.microsoft.com/edge"
		default:
			redirectURL = "https://www.google.com/chrome/"
		}

		http.Redirect(w, r, redirectURL, http.StatusFound)
	}
}

var browserPageHTML = []byte(`<!DOCTYPE html>
<html>
<head>
<meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>提示信息 / Information</title>
<style type="text/css">
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #f5f5f5; color: #333; }
.container { max-width: 1000px; margin: 40px auto; padding: 0 20px; text-align: center; }
.banner { background: #fff; border-radius: 12px; padding: 60px 40px; box-shadow: 0 2px 12px rgba(0,0,0,0.08); }
.icon { font-size: 64px; margin-bottom: 20px; }
h1 { font-size: 24px; margin-bottom: 12px; color: #d4557f; }
p { font-size: 14px; color: #888; margin-bottom: 30px; line-height: 1.6; }
.browsers { display: flex; justify-content: center; gap: 20px; flex-wrap: wrap; }
.browser-card { display: flex; flex-direction: column; align-items: center; padding: 20px 30px; background: #fafafa; border-radius: 8px; text-decoration: none; color: #333; transition: all 0.2s; min-width: 140px; }
.browser-card:hover { background: #f0f0f0; transform: translateY(-2px); box-shadow: 0 4px 12px rgba(0,0,0,0.1); }
.browser-card .bname { font-size: 16px; font-weight: 600; margin-top: 8px; }
.browser-card .bdesc { font-size: 12px; color: #999; margin-top: 4px; }
@media (max-width: 600px) {
  .banner { padding: 30px 20px; }
  .browsers { flex-direction: column; align-items: center; }
}
</style>
</head>
<body>
<div class="container">
  <div class="banner">
    <div class="icon">&#9888;</div>
    <h1>您的浏览器版本过低 / Browser Too Old</h1>
    <p>为了获得更好的浏览体验，请升级到现代浏览器。<br>For the best experience, please upgrade to a modern browser.</p>
    <div class="browsers">
      <a class="browser-card" href="/browser-download/chrome" target="_blank">
        <span style="font-size:40px;">&#x1F310;</span>
        <span class="bname">Chrome</span>
        <span class="bdesc">谷歌浏览器</span>
      </a>
      <a class="browser-card" href="/browser-download/firefox" target="_blank">
        <span style="font-size:40px;">&#x1F525;</span>
        <span class="bname">Firefox</span>
        <span class="bdesc">火狐浏览器</span>
      </a>
      <a class="browser-card" href="/browser-download/ie" target="_blank">
        <span style="font-size:40px;">&#x1F4BB;</span>
        <span class="bname">Edge</span>
        <span class="bdesc">微软 Edge</span>
      </a>
    </div>
  </div>
  <p style="margin-top: 20px; font-size: 12px; color: #aaa;">
    您的浏览器信息 / Your browser information：<script>document.write(navigator.userAgent);</script>
  </p>
</div>
</body>
</html>`)
