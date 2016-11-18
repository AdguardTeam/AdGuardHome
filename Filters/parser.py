import urllib2, datetime, mmap, re

## FUNCTION ##
def is_domain_rule(rule):
    point_idx = rule.find('.')
    if point_idx == -1:
        return False
    question_idx = rule.find('?', point_idx);    
    slash_idx = rule.find('/', point_idx)
    if slash_idx == -1 and question_idx == -1:
        return True
    replace_idx =  slash_idx if slash_idx != -1 else question_idx
    tail = rule[replace_idx:]
    return len(tail) <= 2

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

def is_rule_not_exclusion(rule, exclusions):
    for line in exclusions:
        if line in rule and line != '':  
            return False        
    return True
  
def write_rule(rule, f):
    if is_domain_rule(rule):
        f.writelines(rule + '\n')

def save_url_rule(line, exclusions, f):
    url = line.replace('url', '').strip() 
    for rule in get_content(url):
        if is_rule_not_exclusion(rule, exclusions):      
            if rule.find('$') != -1:
                idx = rule.find('$');
                write_rule(rule[:idx], f)        
            else: 
                write_rule(rule, f)

def save_file_rule(line, f):
    file_name = line.replace('file', '').strip()
    with open(file_name, 'r') as rf:
        for rule in rf:
            f.writelines(rule)

## MAIN ##
exclusions = open('exclusions.txt', 'r').read().split('\n')
with open('filter.template', 'r') as tmpl:    
   with open('filter.txt', 'w') as f:
       for line in tmpl:
            if line.startswith('!'):
                save_comment(line, f)
            if line.startswith('url'):
                save_url_rule(line, exclusions, f)
            if line.startswith('file'):
                save_file_rule(line, f)              
                    
