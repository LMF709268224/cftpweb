git pull
bash image_build.sh
kubectl delete pods -l app=adminbff -n cftp-test
sleep 5

