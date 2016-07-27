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

def save_url_rule(url, f):
    url = line.replace('url', '').strip() 
    for rule in get_content(url):        
        if rule.find('^') != -1:
            idx = rule.find('^')         
            f.writelines(rule[:idx] + '\n')
        elif rule.find('$') != -1:
            idx = rule.find('$');
            f.writelines(rule[:idx] + '\n')        
        else: 
            f.writelines(rule + '\n')    

def save_file_rule(line, f):
    file_name = line.replace('file', '').strip()
    with open(file_name, 'r') as rf:
        for rule in rf:
            f.writelines(rule)

with open('filter.template', 'r') as tmpl:    
    with open('filter.txt', 'w') as f:
       for line in tmpl:
            if line.startswith('!'):
                save_comment(line, f)
            if line.startswith('url'):
                save_url_rule(line, f)
            if line.startswith('file'):
                save_file_rule(line, f)
                
                    
