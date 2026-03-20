import { onFinish } from "./loginjs";
import { useEffect } from 'react';

const YourComponent = () => {
  useEffect(() => {
    // 输入框获取焦点时的样式处理函数
    function addcl() {
      let parent = this.parentNode.parentNode;
      parent.classList.add("focus");
    }

    // 输入框失去焦点时的样式处理函数
    function remcl() {
      let parent = this.parentNode.parentNode;
      if (this.value === "") {
        parent.classList.remove("focus");
      }
    }

    // 获取DOM元素
    const sign_in_btn = document.querySelector("#sign-in-btn");
    const sign_up_btn = document.querySelector("#sign-up-btn");
    const container = document.querySelector(".container");
    const loginBtn = document.getElementById("login-btn");
    const usernameDiv = document.querySelector(".input-divname");
    const passwordDiv = document.querySelector(".input-divpass");
    const inputs = document.querySelectorAll(".input");
    const passwordInput = document.getElementById('password');
    const usernameInput = document.getElementById('username');
    // 输入框的焦点和失焦事件监听
    inputs.forEach((input) => {
      input.addEventListener("focus", addcl);
      input.addEventListener("blur", remcl);
    });

    // 其他事件监听
    sign_up_btn.addEventListener("click", () => {
      container.classList.add("sign-up-mode");
    });

    sign_in_btn.addEventListener("click", () => {
      container.classList.remove("sign-up-mode");
    });
    passwordInput.addEventListener('input', () => {
      if (passwordInput.value === '') {
        passwordDiv.classList.add('error');
      } else {
        passwordDiv.classList.remove('error');
        passwordInput.setCustomValidity('');
      }
    });
    usernameInput.addEventListener('input', () => {
      if (usernameInput.value === '') {
        usernameDiv.classList.add('error');
      } else {
        usernameDiv.classList.remove('error');
        usernameInput.setCustomValidity('');
      }
    });
    loginBtn.addEventListener("click", function (event) {
      event.preventDefault();

      // 获取用户名和密码输入框
      const usernameInput = document.getElementById('username');
      const passwordInput = document.getElementById('password');

      // 验证用户名和密码是否为空
      if (usernameInput.value !== "" && passwordInput.value !== "") {
        // 构建表单数据对象
        const formData = { account: usernameInput.value, password: passwordInput.value };
        // 假设onFinish是来自"./loginjs"的函数
        onFinish(event, JSON.stringify(formData));
      } else if (passwordInput.value === "") {
        passwordInput.setCustomValidity("请输入密码");
        passwordInput.reportValidity();
        passwordDiv.classList.add("error");
      } else {
        passwordInput.setCustomValidity("");
        passwordDiv.classList.remove("error");
      }
      if (usernameInput.value === "") {
        usernameInput.setCustomValidity("请输入用户名");
        usernameInput.reportValidity();
        usernameDiv.classList.add("error");
      } else {
        usernameInput.setCustomValidity("");
        usernameDiv.classList.remove("error");
      }
    });

    return () => {
      // 清理函数：移除事件监听器
      inputs.forEach((input) => {
        input.removeEventListener("focus", addcl);
        input.removeEventListener("blur", remcl);
      });
      // 移除其他事件监听器
    };
  }, []); // 空依赖数组表示它在初始渲染后只运行一次

  // 渲染组件的JSX
  return <div></div>
};

export default YourComponent;
