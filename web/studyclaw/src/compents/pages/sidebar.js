import { useEffect } from 'react';

const Sidebar = () => {
  useEffect(() => {
    const handleClick = () => {
      document.querySelector('#menu-toggle:checked').checked = false;
      // 执行其他的操作
    };

    
    const bindEventHandlers = () => {
      const link1 = document.getElementById('user-link1');
      const link2 = document.getElementById('user-link2');
      const link3 = document.getElementById('user-link3');
      const link4 = document.getElementById('user-link4');
      const link5 = document.getElementById('user-link5');

      if (link1) link1.addEventListener('click', handleClick);
      if (link2) link2.addEventListener('click', handleClick);
      if (link3) link3.addEventListener('click', handleClick);
      if (link4) link4.addEventListener('click', handleClick);
      if (link5) link5.addEventListener('click', handleClick);
    };

    const unbindEventHandlers = () => {
      const link1 = document.getElementById('user-link1');
      const link2 = document.getElementById('user-link2');
      const link3 = document.getElementById('user-link3');
      const link4 = document.getElementById('user-link4');
      const link5 = document.getElementById('user-link5');

      if (link1) link1.removeEventListener('click', handleClick);
      if (link2) link2.removeEventListener('click', handleClick);
      if (link3) link3.removeEventListener('click', handleClick);
      if (link4) link4.removeEventListener('click', handleClick);
      if (link5) link5.removeEventListener('click', handleClick);
    };

    const handleResize = () => {
      if (window.innerWidth < 1360) {
        bindEventHandlers();
      } else {
        unbindEventHandlers();
      }
    };

    handleResize(); // 初始化时检查窗口宽度

    window.addEventListener('resize', handleResize);

    return () => {
      window.removeEventListener('resize', handleResize);
      unbindEventHandlers();
    };
  }, []);

  // 渲染组件的 JSX
  return <div></div>;
};

export default Sidebar;