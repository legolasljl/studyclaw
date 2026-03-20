## 可执行文件运行

+ 本地访问[Releases]([Release](https://github.com/legolasljl/studyclaw/releases)) ,查找对应版本并复制链接
+ 使用wget下载对应版本压缩包
+ > tar -xzvf studyclaw_linux_amd64.tar.gz
+ 运行 ```./studyclaw --init```,首次运行会生成默认配置文件
+ 使用vim修改对应配置文件，linux建议使用tg模式运行，详情配置参考[配置](../config.md),推送方式查看[push](../push.md)
+ 再次运行即可

### 一键安装脚本
```shell
wget  https://raw.githubusercontent.com/sjkhsl/studyclaw/main/docs/studyclaw_install.py && python3 studyclaw_install.py
```

## docker运行

```
docker run --name studyclaw --restart always -d -p 8080:8080 -v /etc/studyclaw/:/opt/config/  sjkhsl/studyclaw:latest
```
各个参数的含义：
+ **--restart always 容器总是自动重启
+ **--name studyclaw** 运行的容器的名称，可以根据自己实际情况进行修改
+ **-p 8080:8080** 将容器内部的8080端口映射到容器外面，前面是宿主机的端口，就是网页上访问的端口，后面是容器内部需要运行的端口，对应配置文件内web配置的端口就好
+ **-v /etc/studyclaw/:/opt/config/** 将容器内的/opt/config/目录映射到宿主机的/etc/studyclaw/目录，可根据实际情况修改前面宿主机路径，映射后对应的config.yml配置文件位置就在该目录下
+ **sjkhsl/studyclaw:latest**镜像名称和镜像的版本，latest代表开发中的最新版本

## docker-compose运行

```shell
wget https://raw.githubusercontent.com/sjkhsl/studyclaw/main/docker-compose.yml
docker-compose up -d
```

## 二种运行方式的区别

+ #### 可执行文件运行

    可执行文件运行节省存储空间，拥有更低的占用，但是可能会存在浏览器依赖安装的问题，适合拥有一定linux基础的用户使用
    ，如果系统为debian11用户，可以参考DockerFile文件中的依赖安装语句执行即可，centos用户推荐使用docker.
+ #### docker运行
    docker运行不需要解决依赖问题，但是可能面临更高的运行占用，建议使用docker控制内存占用
