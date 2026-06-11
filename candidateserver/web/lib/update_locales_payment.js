const fs = require('fs');

function updateFile(filePath, isZh) {
  let content = fs.readFileSync(filePath, 'utf8');

  const paymentAdditions = isZh ? `
  paymentReturnHandler: {
    title: "支付结果",
    purchaseSuccess: "购买成功，认证列表已刷新。",
    unlockSuccess: "解锁成功，认证列表已刷新。",
    cancelled: "支付已取消，你可以稍后继续处理订单。",
    failed: "支付失败，请稍后重试或联系管理员。",
    inProgressDesc: "支付尚未完成，订单仍在处理中。请回到认证中心继续支付或重新检查状态。",
    unknownDesc: "支付流程已返回，但没有收到支付结果。请回到认证中心重新检查订单状态。"
  },` : `
  paymentReturnHandler: {
    title: "Payment Result",
    purchaseSuccess: "Purchase successful. The course list has been refreshed.",
    unlockSuccess: "Unlock successful. The course list has been refreshed.",
    cancelled: "Payment cancelled. You can continue the order later.",
    failed: "Payment failed. Please try again later or contact support.",
    inProgressDesc: "Payment is not complete. Your order is still processing. Please return to the certification center to continue or recheck the status.",
    unknownDesc: "Payment process returned, but no result was received. Please return to the certification center to recheck the order status."
  },`;

  content = content.replace(/paymentReturnHandler: \{[\s\S]*?\},/, paymentAdditions);
  fs.writeFileSync(filePath, content);
}

updateFile('d:/Go/src/cftpweb/candidateserver/web/lib/locales/zh.ts', true);
updateFile('d:/Go/src/cftpweb/candidateserver/web/lib/locales/en.ts', false);
