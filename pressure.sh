go test -c #生成可执行文件
PKG=$(basename $(pwd)) # 获取当前路径的最后一个名字，即为文件夹的名字
echo $PKG

count=50
while  [ $count -gt 0 ]; do
    export GOMAXPROCS=$[ 1 + $[ RANDOM % 128 ]] # 随机的GOMAXPROCS
    ./$PKG.test $@ 2>&1 # $@代表可以加入参数 2>&1代表错误输出到控制台
    ((count--))
done