<script setup lang="ts">
import "vditor/dist/index.css";
import Vditor from "vditor";
import { useDark } from "@pureadmin/utils";
import { useIntervalFn } from "@vueuse/core";
import { onMounted, ref, watch, toRaw, onUnmounted } from "vue";

/**
 * Vditor Markdown 编辑器（移植自上游 vue-pure-admin 的
 * src/views/markdown/components/Vditor.vue）。
 * - `v-model` 双向绑定 Markdown 源文本；
 * - 跟随亮/暗主题自动 `setTheme`（初始挂载也按当前主题渲染）；
 * - `cache.enable=false`：不落本地缓存，避免同一页面多个语言 Tab 互相串内容；
 * - `onUnmounted` 销毁实例，释放 DOM 与轮询。
 */
defineOptions({ name: "MarkdownEditor" });

const props = withDefaults(
  defineProps<{
    modelValue?: string;
    /** 透传给 Vditor 的原始 options（cache/fullscreen 由本组件固定） */
    options?: Record<string, any>;
  }>(),
  { modelValue: "", options: () => ({}) }
);

const emit = defineEmits<{
  "update:modelValue": [string];
  after: [Vditor];
  focus: [string];
  blur: [string];
  esc: [string];
  ctrlEnter: [string];
  select: [string];
}>();

const { isDark } = useDark();
const editor = ref<Vditor | null>(null);
const markdownRef = ref<HTMLElement | null>(null);

onMounted(() => {
  editor.value = new Vditor(markdownRef.value as HTMLElement, {
    ...props.options,
    value: props.modelValue,
    cache: {
      enable: false
    },
    fullscreen: {
      index: 10000
    },
    after() {
      emit("after", toRaw(editor.value) as Vditor);
    },
    input(value: string) {
      emit("update:modelValue", value);
    },
    focus(value: string) {
      emit("focus", value);
    },
    blur(value: string) {
      emit("blur", value);
    },
    esc(value: string) {
      emit("esc", value);
    },
    ctrlEnter(value: string) {
      emit("ctrlEnter", value);
    },
    select(value: string) {
      emit("select", value);
    }
  });
});

watch(
  () => props.modelValue,
  newVal => {
    if (newVal !== editor.value?.getValue()) {
      editor.value?.setValue(newVal);
    }
  }
);

watch(
  () => isDark.value,
  newVal => {
    const { pause } = useIntervalFn(() => {
      if (editor.value?.vditor) {
        newVal
          ? editor.value.setTheme("dark", "dark", "rose-pine")
          : editor.value.setTheme("classic", "light", "github");
        pause();
      }
    }, 20);
  },
  { immediate: true }
);

onUnmounted(() => {
  const editorInstance = editor.value;
  if (!editorInstance) return;
  try {
    editorInstance?.destroy?.();
  } catch (error) {
    console.log(error);
  }
});
</script>

<template>
  <div ref="markdownRef" class="markdown-editor" />
</template>

<style scoped>
.markdown-editor {
  width: 100%;
}

.markdown-editor :deep(.vditor) {
  width: 100%;
}
</style>
