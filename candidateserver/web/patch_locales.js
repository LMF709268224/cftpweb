const fs = require('fs');

function patchZh() {
  let content = fs.readFileSync('d:/Go/src/cftpweb/candidateserver/web/lib/locales/zh.ts', 'utf8');

  const benefits = `
    basicBenefits: ["基础课程访问", "社区论坛参与", "月度通讯订阅"],
    certifiedBenefits: ["全部课程访问", "Webinar 录播", "资料下载", "CFtP 称号使用", "课程折扣 20%"],
    premiumBenefits: ["全部持证会员权益", "1对1 导师辅导", "线下活动优先", "企业培训折扣", "定制学习计划"],`;
  content = content.replace(/(introDesc:\s*".*?",)/, "$1" + benefits);

  const newSections = `
  todoList: {
    items: "项",
    noPendingTasks: "暂无待处理事项"
  },
  recordsPage: {
    title: "档案中心",
    subtitle: "管理您的学历、证书和工作经历档案",
    uploadNew: "上传新档案",
    myRecords: "我的档案",
    noRecords: "暂无档案记录",
    noRecordsDesc: "档案上传和审核接口接入后，这里会展示真实的学历、证书和工作经历记录。",
    verified: "已认证",
    pending: "审核中",
    rejected: "已驳回"
  },
  loginPage: {
    and: "和",
    login: "登录",
    welcomeBack: "欢迎回来",
    loginPrompt: "请使用统一身份认证系统 (SSO) 登录您的账号",
    connecting: "正在安全连接...",
    termsPrefix: "点击登录即表示您同意我们的",
    termsOfService: "服务条款",
    privacyPolicy: "隐私政策",
    enterpriseAuth: "企业级统一认证",
    sloganTitleLine1: "安全、高效的",
    sloganTitleLine2: "下一代通行证",
    sloganDesc: "基于 Casdoor 驱动，为您提供金融级别的安全防护、无缝单点登录体验与全球边缘加速接入。",
    multiDevice: "多端协同",
    multiDeviceDesc: "一次登录，畅享全生态微服务矩阵，告别重复认证。",
    globalNetwork: "全球网络",
    globalNetworkDesc: "智能感知网络链路，为您分配最近的认证节点。"
  },
  paymentReturnHandler: {
    title: "支付结果",
    purchaseSuccess: "购买成功，认证列表已刷新。",
    unlockSuccess: "解锁成功，认证列表已刷新。",
    cancelled: "支付已取消，你可以稍后继续处理订单。",
    failed: "支付失败，请稍后重试或联系管理员。",
    inProgressDesc: "支付尚未完成，订单仍在处理中。请回到认证中心继续支付或重新检查状态。",
    unknownDesc: "支付流程已返回，但没有收到支付结果。请回到认证中心重新检查订单状态。"
  },
  purchaseDialog: {
    title: "认证购买状态",
    checking: "正在检查你是否可以购买或解锁...",
    canPurchaseTitle: "可以购买认证",
    canPurchaseDesc: "你已满足购买条件，可以创建购买订单并查看价格。",
    canUnlockTitle: "需要先解锁认证",
    canUnlockDesc: "这个认证需要先解锁。解锁完成后，系统会重新检查，然后才可以购买认证。",
    blockedTitle: "暂时不能购买或解锁",
    blockedDesc: "请先处理下面的阻塞原因。",
    blockersTitle: "阻塞原因",
    requiredItems: "需要完成",
    missingQualification: "缺少解锁资格",
    alreadyPurchased: "你已经购买过该认证",
    inProgressPurchase: "已有未完成的购买订单",
    inProgressPurchaseDesc: "你已有未完成订单，可以继续查看价格并完成支付。",
    pipelineNotFound: "该认证已不可用",
    unknownBlocker: "暂时不能继续",
    createPurchaseOrder: "创建购买订单",
    createUnlockOrder: "创建解锁订单",
    refreshEligibility: "重新检查状态",
    pricePreviewTitle: "价格预览",
    pricePreviewFailed: "暂时无法获取价格预览，请稍后重试。价格未确认前不能发起支付。",
    retryPreview: "重新获取价格",
    orderCreated: "订单已创建",
    activeOrder: "未完成订单",
    unlockCompleted: "解锁已完成，请重新检查购买状态。",
    subtotal: "原价",
    discount: "优惠",
    tax: "税费",
    total: "应付合计",
    stripe: "Stripe 在线支付",
    bank: "银行转账",
    payNow: "去支付",
    embeddedCheckoutTitle: "请在下方完成支付",
    embeddedCheckoutDesc: "支付会话已创建。支付完成后，Stripe 会返回认证中心并刷新订单状态。",
    embeddedCheckoutLoading: "正在加载 Stripe 支付组件...",
    stripePublishableKeyMissing: "缺少 Stripe Publishable Key，请配置 STRIPE_PUBLISHABLE_KEY。",
    stripeEmbeddedFailed: "支付组件加载失败，请刷新后重试。",
    paymentSessionFailed: "支付会话创建失败，请稍后重试。",
    unsupportedPaymentKey: "暂不支持的支付凭证类型，请稍后重试。",
    purchaseCompleted: "购买成功，认证已开通。",
    purchaseFailed: "购买失败，请稍后重试或联系管理员。",
    unlockFailed: "解锁失败，请稍后重试或联系管理员。"
  }`;

  content = content.replace(/\n}\s*export type AppTranslations[\s\S]*$/, ",\n" + newSections + "\n}\n\nexport type AppTranslations = typeof zh;\n");
  fs.writeFileSync('d:/Go/src/cftpweb/candidateserver/web/lib/locales/zh.ts', content);
}

function patchEn() {
  let content = fs.readFileSync('d:/Go/src/cftpweb/candidateserver/web/lib/locales/en.ts', 'utf8');

  const benefits = `
    basicBenefits: ["Basic courses access", "Community forum access", "Monthly newsletter"],
    certifiedBenefits: ["All courses access", "Webinar recordings", "Materials download", "CFtP designation usage", "20% course discount"],
    premiumBenefits: ["All certified benefits", "1-on-1 mentorship", "Priority offline events", "Corporate training discounts", "Custom study plans"],`;
  content = content.replace(/(introDesc:\s*".*?",)/, "$1" + benefits);

  const newSections = `
  todoList: {
    items: "Items",
    noPendingTasks: "No pending tasks"
  },
  recordsPage: {
    title: "Records Center",
    subtitle: "Manage your education, certification, and work experience records",
    uploadNew: "Upload New Record",
    myRecords: "My Records",
    noRecords: "No records yet",
    noRecordsDesc: "When the upload API is connected, your records will be displayed here.",
    verified: "Verified",
    pending: "Pending",
    rejected: "Rejected"
  },
  loginPage: {
    and: "and",
    login: "Login",
    welcomeBack: "Welcome Back",
    loginPrompt: "Please log in using the Single Sign-On (SSO) system",
    connecting: "Connecting securely...",
    termsPrefix: "By clicking login, you agree to our",
    termsOfService: "Terms of Service",
    privacyPolicy: "Privacy Policy",
    enterpriseAuth: "Enterprise Unified Auth",
    sloganTitleLine1: "Secure and Efficient",
    sloganTitleLine2: "Next-Gen Passport",
    sloganDesc: "Powered by Casdoor, providing financial-grade security, seamless SSO, and global edge acceleration.",
    multiDevice: "Multi-Device Sync",
    multiDeviceDesc: "Log in once to access the entire microservice matrix. Say goodbye to repeated logins.",
    globalNetwork: "Global Network",
    globalNetworkDesc: "Intelligent routing to allocate the nearest authentication node."
  },
  paymentReturnHandler: {
    title: "Payment Result",
    purchaseSuccess: "Purchase successful. The certification list has been refreshed.",
    unlockSuccess: "Unlock successful. The certification list has been refreshed.",
    cancelled: "Payment cancelled. You can continue the order later.",
    failed: "Payment failed. Please try again later or contact support.",
    inProgressDesc: "Payment is not complete. Your order is still processing. Please return to the certification center to continue or recheck the status.",
    unknownDesc: "Payment process returned, but no result was received. Please return to the certification center to recheck the order status."
  },
  purchaseDialog: {
    title: "Purchase Status",
    checking: "Checking whether you can purchase or unlock this certification...",
    canPurchaseTitle: "Ready to purchase",
    canPurchaseDesc: "You meet the requirements. Create an order to preview the price.",
    canUnlockTitle: "Unlock required first",
    canUnlockDesc: "This certification must be unlocked before purchase. After unlocking, the system will recheck eligibility.",
    blockedTitle: "Action unavailable",
    blockedDesc: "Please resolve the blockers below first.",
    blockersTitle: "Blockers",
    requiredItems: "Required",
    missingQualification: "Missing unlock qualification",
    alreadyPurchased: "You have already purchased this certification",
    inProgressPurchase: "Purchase already in progress",
    inProgressPurchaseDesc: "You have an unfinished order. You can continue to review the price and complete payment.",
    pipelineNotFound: "This certification is no longer available",
    unknownBlocker: "Unable to continue",
    createPurchaseOrder: "Create purchase order",
    createUnlockOrder: "Create unlock order",
    refreshEligibility: "Recheck status",
    pricePreviewTitle: "Price preview",
    pricePreviewFailed: "Price preview is temporarily unavailable. Payment cannot be initiated until the price is confirmed.",
    retryPreview: "Retry price preview",
    orderCreated: "Order created",
    activeOrder: "Unfinished order",
    unlockCompleted: "Unlock completed. Recheck purchase status.",
    subtotal: "Subtotal",
    discount: "Discount",
    tax: "Tax",
    total: "Total due",
    stripe: "Stripe online payment",
    bank: "Bank transfer",
    payNow: "Pay now",
    embeddedCheckoutTitle: "Complete payment below",
    embeddedCheckoutDesc: "Payment session created. After payment, Stripe will return you to the center and refresh the order status.",
    embeddedCheckoutLoading: "Loading Stripe checkout...",
    stripePublishableKeyMissing: "Missing Stripe publishable key. Please configure STRIPE_PUBLISHABLE_KEY.",
    stripeEmbeddedFailed: "Failed to load payment component. Please refresh and try again.",
    paymentSessionFailed: "Failed to create payment session. Please try again later.",
    unsupportedPaymentKey: "Unsupported payment credential type. Please try again later.",
    purchaseCompleted: "Purchase successful. The certification is now active.",
    purchaseFailed: "Purchase failed. Please try again later or contact support.",
    unlockFailed: "Unlock failed. Please try again later or contact support."
  }`;

  content = content.replace(/}\s*$/, ",\n" + newSections + "\n}");
  fs.writeFileSync('d:/Go/src/cftpweb/candidateserver/web/lib/locales/en.ts', content);
}

patchZh();
patchEn();
