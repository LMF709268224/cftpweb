import sys

file_path = r'd:\Go\src\cftpweb\adminweb\vue-web\src\pages\LmsPage.vue'
with open(file_path, 'r', encoding='utf-8') as f:
    content = f.read()

target1 = '''            <div class="flex-1 space-y-5 overflow-y-auto p-5">
              <div class="grid gap-4 lg:grid-cols-3">'''
repl1 = '''            <div class="flex-1 space-y-5 overflow-y-auto p-5">
              <div v-if="quizDialogMode !== 'create'" class="mb-4 flex gap-4 border-b border-slate-200">
                <button :class="quizActiveTab === 'basic' ? 'border-blue-500 text-blue-600' : 'border-transparent text-slate-500'" class="border-b-2 px-1 pb-2 font-bold transition-colors" type="button" @click="quizActiveTab = 'basic'">{{ (copy as any).basicInfo || '基本信息' }}</button>
                <button :class="quizActiveTab === 'prerequisites' ? 'border-blue-500 text-blue-600' : 'border-transparent text-slate-500'" class="border-b-2 px-1 pb-2 font-bold transition-colors" type="button" @click="quizActiveTab = 'prerequisites'">{{ (copy as any).prerequisites || '前置条件' }}</button>
              </div>
              <div v-show="quizActiveTab === 'basic'">
                <div class="grid gap-4 lg:grid-cols-3">'''

# Search for the exact end block
idx = content.find("</section>\n                </div>\n              <div v-if=\"quizActiveTab === 'prerequisites'")
if idx != -1:
    content = content[:idx] + "</section>\n              </div>\n              <div v-if=\"quizActiveTab === 'prerequisites'" + content[idx + len("</section>\n                </div>\n              <div v-if=\"quizActiveTab === 'prerequisites'"):]

if target1 in content and idx != -1:
    content = content.replace(target1, repl1)
    with open(file_path, 'w', encoding='utf-8') as f:
        f.write(content)
    print("Patched successfully")
else:
    print("Could not find targets")

