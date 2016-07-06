import urllib2
import datetime

def date_now():
    return datetime.datetime.now().strftime("%Y-%m-%d %H:%M:%S") 

def get_content(url):
    r = urllib2.urlopen(url)    
    return r.read().split('\n')

def save_comment(comment, f):
    idx = comment.find('%timestamp%')
    if idx != -1:
       comment = comment[:idx] + date_now() + '\n'
    f.writelines(comment)    

def save_rule(url, f):
    url = line.replace('include', '').strip() 
    for rule in get_content(url):
        rule = rule.replace('^', '')
        idx = rule.find('$third-party')        
        if idx != -1:
            f.writelines(rule[:idx] + '\n')
        else: 
            f.writelines(rule + '\n')    

with open('filter.template', 'r') as tmpl:    
    with open('filter.txt', 'w') as f:
       for line in tmpl:
            if line.startswith('!'):
                save_comment(line, f)
            if line.startswith('include'):
                save_rule(line, f)
                
                    
