# 首頁設計文件 — Nintendo Switch UI 風格

**日期：** 2026-03-24
**範圍：** 首頁 Dashboard + 全域 Layout

---

## 設計風格

Nintendo Switch 主選單風格：
- 深色背景（深藍黑）
- 圓角白色卡片
- 高對比粗體數字
- Switch 紅主色調
- 乾淨、現代、直覺

---

## 色彩系統

| 用途 | 色碼 |
|------|------|
| 主色（Switch 紅） | `#E4000F` |
| 背景 | `#1a1a2e` |
| 卡片背景 | `#ffffff` |
| 文字主色 | `#1a1a1a` |
| 文字次色 | `#6b7280` |
| 體重色條 | `#E4000F` |
| 體脂率色條 | `#F97316` |
| 肌肉率色條 | `#3B82F6` |
| 內臟脂肪色條 | `#8B5CF6` |

---

## 全域 Layout（`+layout.svelte`）

### Top Nav
- 背景：Switch 紅 `#E4000F`
- 高度：`h-14`
- 左側：「HEALTH TRACKER」白色粗體（`font-black tracking-widest`）
- 右側：導覽連結
  - 「體位數據」— 可點，連結 `/body-metrics`
  - 「睡眠」— disabled（`opacity-40 cursor-not-allowed`）
  - 「活動」— disabled

### 主內容區
- 背景：`#1a1a2e`
- padding：`px-6 py-8`
- 最大寬度：`max-w-5xl mx-auto`

---

## 首頁（`+page.svelte`）

### 頁面標題區
- 標題：「TODAY」白色大字（`text-2xl font-black tracking-widest`）
- 副標題：今天日期（`text-gray-400 text-sm`）

### 數據卡片區

四張卡片橫向排列（`grid grid-cols-4 gap-4`），手機版 `grid-cols-2`。

**單張卡片結構：**
```
┌──────────────────┐
│ ████  ← 4px 色條 │
│                  │
│  emoji  （text-3xl） │
│                  │
│  72.5  kg        │  ← 數值 text-4xl font-black，單位同行底部對齊 text-base text-gray-400
│                  │
│  體重            │  ← 標籤 text-xs text-gray-500
└──────────────────┘
```

| 卡片 | emoji | 顯示格式 | 色條 |
|------|-------|---------|------|
| 體重 | ⚖️ | `72.5 kg` | `#E4000F` |
| 體脂率 | 🔥 | `18.2 %` | `#F97316` |
| 肌肉率 | 💪 | `35.2 %` | `#3B82F6` |
| 內臟脂肪 | 📊 | `Lv. 8` | `#8B5CF6` |

**互動：**
- hover：`-translate-y-1 shadow-xl`（transition 150ms）
- 無資料時數值顯示 `—`（接 API 後實作，mock 階段暫不處理）

### 趨勢圖區

深色卡片（`bg-white/5 rounded-2xl p-6`）：
- 標題：「TRENDS」白色粗體
- 副標題：「近 30 天」灰色小字
- 圖表佔位：`h-48 bg-white/5 rounded-xl`（Milestone 1 正式接 layerchart）
- 佔位文字：「圖表載入中...」置中灰色

---

## 檔案規劃

```
src/
├── routes/
│   ├── +layout.svelte   ← Top Nav + 主內容區背景
│   └── +page.svelte     ← 今日數據卡片 + 趨勢圖佔位
```

---

## 未來擴充（不在本次範圍）

- 導覽列「睡眠」「活動」在 Milestone 2 後解鎖
- 趨勢圖在接 API 後替換佔位區塊
- 卡片數值在接 API 後替換 mock 數值
