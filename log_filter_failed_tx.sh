while read line
do
  echo "$line" | grep 'failed to execute message' >> ./filtered_first_node_logs.txt
done < mypipe