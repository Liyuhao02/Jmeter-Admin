<template>
  <div class="main-layout">
    <a class="skip-link" href="#main-content">跳到主内容</a>
    <!-- 顶部导航栏 -->
    <header class="top-header">
      <div class="header-content">
        <!-- 左侧 Logo -->
        <div class="logo">
          <div class="logo-icon">
            <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z" fill="url(#logoGradient)" />
              <defs>
                <linearGradient id="logoGradient" x1="3" y1="2" x2="21" y2="22" gradientUnits="userSpaceOnUse">
                  <stop stop-color="#0066ff" />
                  <stop offset="1" stop-color="#00aaff" />
                </linearGradient>
              </defs>
            </svg>
          </div>
          <span class="logo-text">JMeter Admin</span>
        </div>

        <!-- 右侧 Tab 导航 -->
        <nav class="tab-nav" aria-label="主导航">
          <router-link
            v-for="tab in tabs"
            :key="tab.path"
            :to="tab.path"
            class="tab-item"
            :class="{ active: isActive(tab.path) }"
            :aria-current="isActive(tab.path) ? 'page' : undefined"
          >
            <component :is="tab.icon" class="tab-icon" />
            <span>{{ tab.name }}</span>
          </router-link>
        </nav>
      </div>
    </header>

    <!-- 主内容区域 -->
    <main id="main-content" class="main-content" tabindex="-1">
      <div
        class="content-wrapper"
        :class="{
          'content-wrapper--wide': isScriptEditPage,
          'content-wrapper--execution': isExecutionPage || isScriptExecutePage
        }"
      >
        <router-view v-slot="{ Component }">
          <transition name="page-fade" mode="out-in">
            <Suspense>
              <component :is="Component" />
              <template #fallback>
                <div class="route-loading-shell">
                  <div class="route-loading-badge">LOADING</div>
                  <div class="route-loading-title">页面资源加载中</div>
                  <div class="route-loading-desc">正在按需加载当前页面模块，这样首屏会更轻，切页也更稳。</div>
                </div>
              </template>
            </Suspense>
          </transition>
        </router-view>
      </div>
    </main>

    <!-- 底部版本信息 -->
    <footer class="footer">
      <span>JMeter Admin v1.0.0</span>
    </footer>
  </div>
</template>

<script setup>
import { computed, Suspense } from 'vue'
import { useRoute } from 'vue-router'
import { Document, Monitor, DataLine } from '@element-plus/icons-vue'

const route = useRoute()

const tabs = [
  { name: '脚本管理', path: '/scripts', icon: Document },
  { name: 'Slave管理', path: '/slaves', icon: Monitor },
  { name: '执行记录', path: '/executions', icon: DataLine }
]

const isActive = (path) => {
  return route.path.startsWith(path)
}

const isScriptEditPage = computed(() => {
  return /^\/scripts\/[^/]+\/edit$/.test(route.path)
})

const isExecutionPage = computed(() => {
  return /^\/executions(\/.*)?$/.test(route.path)
})

const isScriptExecutePage = computed(() => {
  return /^\/scripts\/[^/]+\/execute$/.test(route.path)
})
</script>

<style scoped lang="scss">
.main-layout {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  background:
    radial-gradient(circle at top center, rgba(56, 189, 248, 0.08), transparent 24%),
    radial-gradient(circle at bottom right, rgba(14, 165, 233, 0.06), transparent 20%),
    #0a0e17;
}

.skip-link {
  position: fixed;
  top: 12px;
  left: 16px;
  z-index: 200;
  padding: 10px 14px;
  border-radius: 999px;
  background: #0f172a;
  color: #ffffff;
  text-decoration: none;
  border: 1px solid rgba(54, 191, 250, 0.45);
  box-shadow: 0 14px 28px rgba(0, 0, 0, 0.28);
  transform: translateY(-120%);
  transition: transform 0.2s ease;

  &:focus {
    transform: translateY(0);
    outline: 2px solid rgba(54, 191, 250, 0.75);
    outline-offset: 2px;
  }
}

// 顶部导航栏
.top-header {
  min-height: 74px;
  background: rgba(12, 18, 30, 0.82);
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  backdrop-filter: blur(18px);
  position: sticky;
  top: 0;
  z-index: 100;
  flex-shrink: 0;
  box-shadow: 0 10px 28px rgba(2, 8, 23, 0.22);
}

.header-content {
  max-width: 1820px;
  margin: 0 auto;
  height: 100%;
  padding: 0 clamp(22px, 2.4vw, 38px);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

// Logo 样式
.logo {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 6px 12px 6px 6px;
  border-radius: 16px;
  background: linear-gradient(135deg, rgba(56, 189, 248, 0.08), rgba(15, 23, 42, 0.08));
  border: 1px solid rgba(255, 255, 255, 0.05);
}

.logo-icon {
  width: 34px;
  height: 34px;
  padding: 6px;
  border-radius: 12px;
  background: rgba(15, 23, 42, 0.72);
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.05);

  svg {
    width: 100%;
    height: 100%;
  }
}

.logo-text {
  font-size: 20px;
  font-weight: 700;
  color: #ffffff;
  letter-spacing: 0.2px;
}

// Tab 导航
.tab-nav {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px;
  border-radius: 16px;
  background: rgba(15, 23, 42, 0.44);
  border: 1px solid rgba(255, 255, 255, 0.05);
}

.tab-item {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 42px;
  padding: 0 16px;
  border-radius: 12px;
  font-size: 14px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.68);
  text-decoration: none;
  transition: all 0.25s ease;
  position: relative;
  border: 1px solid transparent;

  &:hover {
    background: rgba(255, 255, 255, 0.06);
    color: rgba(255, 255, 255, 0.92);
    border-color: rgba(255, 255, 255, 0.04);
  }

  &:focus-visible {
    outline: 2px solid rgba(54, 191, 250, 0.85);
    outline-offset: 2px;
    color: #ffffff;
  }

  &.active {
    background: linear-gradient(135deg, rgba(18, 85, 151, 0.38), rgba(10, 86, 148, 0.18));
    color: #ffffff;
    border-color: rgba(56, 189, 248, 0.2);
    box-shadow: inset 0 -1px 0 rgba(56, 189, 248, 0.34);

    &::after {
      display: none;
    }
  }

  .tab-icon {
    width: 16px;
    height: 16px;
  }
}

// 主内容区域
.main-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  background-color: #0a0e17;

  &:focus {
    outline: none;
  }
}

.content-wrapper {
  flex: 1;
  max-width: 1720px;
  width: 100%;
  margin: 0 auto;
  padding: 28px clamp(28px, 3vw, 52px) 36px;
}

.content-wrapper--wide {
  max-width: 1780px;
  padding: 22px clamp(24px, 2.6vw, 44px) 28px;
}

.content-wrapper--execution {
  max-width: 1800px;
  padding: 26px clamp(30px, 3.1vw, 56px) 36px;
}

.route-loading-shell {
  min-height: 360px;
  border-radius: 24px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  background:
    radial-gradient(circle at top left, rgba(0, 170, 255, 0.16), transparent 35%),
    linear-gradient(180deg, rgba(17, 24, 39, 0.98), rgba(10, 14, 23, 0.96));
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  justify-content: center;
  gap: 12px;
  padding: 32px;
  box-shadow: 0 24px 60px rgba(0, 0, 0, 0.24);
}

.route-loading-badge {
  padding: 6px 12px;
  border-radius: 999px;
  background: rgba(0, 170, 255, 0.12);
  color: #36bffa;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.12em;
}

.route-loading-title {
  font-size: 28px;
  font-weight: 700;
  color: #ffffff;
}

.route-loading-desc {
  max-width: 520px;
  font-size: 14px;
  line-height: 1.8;
  color: rgba(255, 255, 255, 0.64);
}

// 底部
.footer {
  padding: 16px clamp(24px, 2.6vw, 40px);
  text-align: center;
  color: rgba(255, 255, 255, 0.34);
  font-size: 12px;
  border-top: 1px solid rgba(255, 255, 255, 0.04);
  flex-shrink: 0;
  background: rgba(10, 14, 23, 0.7);
}

// 页面过渡动画 - 柔和的淡入效果
.page-fade-enter-active {
  transition: opacity 0.3s ease, transform 0.3s ease;
}

.page-fade-leave-active {
  transition: opacity 0.2s ease;
}

.page-fade-enter-from {
  opacity: 0;
  transform: translateY(8px);
}

.page-fade-leave-to {
  opacity: 0;
}

@media (max-width: 1200px) {
  .header-content {
    padding: 0 20px;
  }

  .content-wrapper,
  .content-wrapper--execution,
  .content-wrapper--wide {
    padding-left: 20px;
    padding-right: 20px;
  }
}

@media (max-width: 768px) {
  .top-header {
    min-height: 72px;
  }

  .header-content {
    padding: 10px 14px;
    height: auto;
    flex-wrap: wrap;
  }

  .logo {
    padding-right: 10px;
  }

  .logo-text {
    font-size: 18px;
  }

  .tab-nav {
    width: 100%;
    overflow-x: auto;
    justify-content: flex-start;
    padding: 5px;
  }

  .tab-item {
    flex: 0 0 auto;
  }

  .content-wrapper,
  .content-wrapper--execution,
  .content-wrapper--wide {
    padding: 14px 12px 22px;
  }
}
</style>
