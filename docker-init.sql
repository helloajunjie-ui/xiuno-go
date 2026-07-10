-- 创建应用用户（同时支持 localhost 和 % 连接）
CREATE USER IF NOT EXISTS 'xiuno'@'localhost' IDENTIFIED WITH mysql_native_password BY 'xiuno123';
CREATE USER IF NOT EXISTS 'xiuno'@'%' IDENTIFIED WITH mysql_native_password BY 'xiuno123';
GRANT ALL PRIVILEGES ON xiuno.* TO 'xiuno'@'localhost';
GRANT ALL PRIVILEGES ON xiuno.* TO 'xiuno'@'%';

-- 修改 root 密码认证为 mysql_native_password，支持 TCP 连接
ALTER USER 'root'@'localhost' IDENTIFIED WITH mysql_native_password BY 'root123';
FLUSH PRIVILEGES;
