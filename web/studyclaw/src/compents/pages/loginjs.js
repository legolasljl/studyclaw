import { checkToken, login } from "../../utils/api";
import { Dialog } from "antd-mobile";

export const onFinish = async (event, valueString) => {
event.preventDefault(); // 阻止默认的提交行为

// 确保 valueString 不为 undefined
if (valueString) {
  try {
    const value = JSON.parse(valueString);
    console.log(value);

    const resp = await login(valueString);

    Dialog.alert({ content: resp.message, closeOnMaskClick: false });

    if (resp.success) {
      window.localStorage.setItem("studyclaw_token", resp.data);

      const t = await checkToken();

      console.log(t);

      if (!t) {
        console.log("未登入");
        window.location.href = "/#/login";
      } else {
        if (t.data === 1) {
          console.log("管理員登入");
          sessionStorage.setItem("level", "1");
        } else {
          console.log("不是管理員登入");
          sessionStorage.setItem("level", "2");
        }

        // 异步操作完成后再进行路由导航
        window.location.href = "/#/home";
      }
    }
  } catch (error) {
    console.error("JSON 解析错误：", error);
    // 处理 JSON 解析错误的情况，例如显示错误信息给用户
  }
} else {
  console.error("valueString 为 undefined");
  // 处理 valueString 为 undefined 的情况，例如显示错误信息给用户
}
};
